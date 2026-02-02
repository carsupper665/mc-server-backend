import requests, json, time

BASE_URL = "http://localhost:8080"

class HttpTester:
    def __init__(self, session: requests.Session = None):
        self.ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko)"
        self.session = requests.Session() if session is None else session
        self.is_sess_exist = session is not None
        self.session.headers.update({"Content-Type": "application/json"})
        self.session.headers.update({"User-Agent": self.ua})

    def login_and_verify(self, username: str, password: str):
        status, login_res = self._login(username, password)

        if status == 202:
            print("check your email for verification code")
            code = input("enter verification code: ")
            status, verify_res = self._verification(code)
            if status != 200:
                print(verify_res)
                raise Exception("verification failed")

        if status != 200:
            print(login_res)
            raise Exception("login failed")

        print("login successful")
        return status, login_res

    def _verification(self, code: str):
        url = f"{BASE_URL}/Authentication/verify"
        post_data = {"code": code}
        response = self.session.post(url, json=post_data)
        return self._res_handler(response)

    def _login(self, username: str, password: str):
        url = f"{BASE_URL}/Authentication/login"
        post_data = {"username": username, "password": password}
        response = self.session.post(url, json=post_data)
        return self._res_handler(response)

    def server_rollback(self, file_name: str = "test", server_id: str = "sid"):
        if not self.is_sess_exist:
            print("please login")
            username = input("enter username: ")
            password = input("enter password: ")
            self.login_and_verify(username, password)

        url = f"{BASE_URL}/mc-api/a/recover"
        post_data = {"file_name": file_name, "server_id": server_id}
        response = self.session.post(url, json=post_data)
        return self._res_handler(response)

    def create_server(self, server_type: str, server_ver: str, display_name: str):
        if not self.is_sess_exist:
            print("please login")
            return None, None
        url = f"{BASE_URL}/user/cs"
        response = self.session.post(url,
                    json={"server_type": server_type,
                        "server_ver": server_ver, "display_name": display_name})
        return self._res_handler(response)

    def save_cookies(self, file_path: str = "cookies.json"):
        cookies_dict = self.session.cookies.get_dict()
        with open(file_path, "w", encoding="utf-8") as f:
            json.dump(cookies_dict, f)

    def server(self, server_id: str, action: str):
        v1_api = ["add_mod"]
        if action not in ["start", "stop", "backup", "ls-backup", "usage"] and action not in v1_api:
            raise ValueError("action must be 'start' or 'stop'")
        if not self.is_sess_exist:
            print("please login")
            return None, None
        if action not in v1_api:
            url = f"{BASE_URL}/mc-api/a/{action}/{server_id}"
            if action in ["usage"]:
                response = self.session.get(url)
            else:
                response = self.session.post(url,)
        else:
            url = f"{BASE_URL}/api/v1/server/{action}/{server_id}"
            
        return self._res_handler(response)

    def add_mod(self, server_id: str, mod_id: str, ver_id: str = "", auto_update: bool = True):
        if not self.is_sess_exist:
            print("please login")
            return None, None
        url = f"{BASE_URL}/api/v1/server/mod/add/{server_id}"
        post_data = {
            "mod_id": mod_id,
            "version_id": ver_id,
            "auto_update": auto_update
        }
        response = self.session.post(url, json=post_data)
        return self._res_handler(response)

    def load_cookies(self, file_path: str = "cookies.json"):
        with open(file_path, "r", encoding="utf-8") as f:
            cookies_dict = json.load(f)
        cookies_jar = requests.utils.cookiejar_from_dict(cookies_dict)
        self.session.cookies = cookies_jar
        self.is_sess_exist = True

    def log_out(self):
        url = f"{BASE_URL}/logout"
        response = self.session.post(url)
        return self._res_handler(response)
    
    def list_servers(self):
        if not self.is_sess_exist:
            print("please login")
            return None, None
        url = f"{BASE_URL}/user/myservers"
        response = self.session.get(url)
        return self._res_handler(response)

    def _res_handler(self, response: requests.Response):
        print(response.status_code)
        try:
            json_res = response.json()
            print(json_res)
        except json.decoder.JSONDecodeError:
            print("response is not json")
            print(response.raw)
        return response.status_code, response
    
def login_again(username: str, password: str):
    term = HttpTester()
    term.load_cookies("cookies.json")
    term.login_and_verify(username, password)
    term.save_cookies("cookies.json")

if __name__ == "__main__":
    term = HttpTester()
    term.load_cookies("cookies.json")
    term.create_server("Fabric", "1.20.1", "1.20.1 Test")

    # term.add_mod(server_id='mcsfv-fabric-0269-OID-1', mod_id='AANobbMI') # add-mod-test
    # term.add_mod(server_id='mcsfv-fabric-6308-OID-1', mod_id='AANobbMI')
