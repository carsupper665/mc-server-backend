import argparse
import json
import os
import sys
import threading
import time

import work_flow_test as wf


PALETTE = {
    "bg_base": "#1e1e2e",
    "bg_surface0": "#313244",
    "fg_text": "#cdd6f4",
    "fg_muted": "#6c7086",
    "fg_subtext0": "#a6adc8",
    "rosewater": "#f5e0dc",
    "flamingo": "#f2cdcd",
    "pink": "#f5c2e7",
    "mauve": "#cba6f7",
    "lavender": "#b4befe",
    "maroon": "#eba0ac",
}


def enable_windows_ansi():
    if os.name != "nt":
        return
    try:
        import ctypes

        kernel32 = ctypes.windll.kernel32
        handle = kernel32.GetStdHandle(-11)  # STD_OUTPUT_HANDLE
        if handle in (0, -1):
            return
        mode = ctypes.c_uint()
        if kernel32.GetConsoleMode(handle, ctypes.byref(mode)) == 0:
            return
        kernel32.SetConsoleMode(handle, mode.value | 0x0004)  # ENABLE_VIRTUAL_TERMINAL_PROCESSING
    except Exception:
        return


enable_windows_ansi()


TRANSLATIONS = {
    "zh": {
        "cli_desc": "工作流程 CLI（使用 work_flow_test.py 內建函式）",
        "help_lang": "語言 / Language (zh 或 en)",
        "help_base_url": "覆寫 BASE_URL（例如 http://localhost:8080）",
        "help_cookies": "Cookies 檔案路徑（預設 cookies.json）",
        "help_default_env": "可用環境變數設定預設值：WF_LANG / WF_BASE_URL / WF_COOKIES",
        "help_config_path": "設定檔",
        "help_cfg": "儲存全域選項到設定檔（--lang / --base-url / --cookies）",
        "msg_cfg_saved": "已儲存設定",
        "help_quickref_title": "指令速查",
        "help_global_opts": "全域選項",
        "help_examples": "範例",
        "cmd_login": "登入並保存 cookies",
        "cmd_logout": "登出",
        "cmd_list_servers": "列出伺服器",
        "cmd_create_server": "建立伺服器",
        "cmd_server_action": "伺服器動作",
        "cmd_add_mod": "新增模組",
        "cmd_del_mod": "刪除模組",
        "cmd_toggle_mod": "切換模組啟用/停用",
        "cmd_update_mod": "更新模組",
        "cmd_list_mods": "列出模組",
        "cmd_rollback": "回復備份",
        "help_username": "使用者名稱",
        "help_password": "密碼",
        "help_save_cookies": "登入後儲存 cookies",
        "help_server_type": "伺服器類型（例如 Fabric / Vanilla）",
        "help_server_ver": "伺服器版本（例如 1.20.1）",
        "help_display_name": "顯示名稱",
        "help_server_id": "伺服器 ID",
        "help_action": "操作（start / stop / backup / ls-backup / usage）",
        "help_mod_id": "Modrinth 模組 ID",
        "help_version_id": "模組版本 ID（可空白）",
        "help_no_auto_update": "關閉自動更新",
        "help_file_name": "備份檔名",
        "title_login": "登入",
        "title_logout": "登出",
        "title_list_servers": "伺服器列表",
        "title_create_server": "建立伺服器",
        "title_server_action": "伺服器動作",
        "title_add_mod": "新增模組",
        "title_del_mod": "刪除模組",
        "title_toggle_mod": "切換模組",
        "title_update_mod": "更新模組",
        "title_list_mods": "模組列表",
        "title_rollback": "回復備份",
        "desc_login": "登入並取得 session cookies\n必填：username, password",
        "desc_logout": "登出（需要 cookies）",
        "desc_list_servers": "列出使用者擁有的伺服器",
        "desc_create_server": "建立新的伺服器",
        "desc_server_action": "對伺服器執行動作",
        "desc_add_mod": "新增模組到伺服器",
        "desc_del_mod": "從伺服器移除模組",
        "desc_toggle_mod": "切換模組啟用/停用",
        "desc_update_mod": "更新伺服器模組",
        "desc_list_mods": "列出伺服器模組",
        "desc_rollback": "回復伺服器備份",
    },
    "en": {
        "cli_desc": "Workflow CLI (uses work_flow_test.py functions)",
        "help_lang": "Language (zh or en)",
        "help_base_url": "Override BASE_URL (e.g. http://localhost:8080)",
        "help_cookies": "Cookies file path (default: cookies.json)",
        "help_default_env": "Defaults via env: WF_LANG / WF_BASE_URL / WF_COOKIES",
        "help_config_path": "Config file",
        "help_cfg": "Save global options to config (--lang / --base-url / --cookies)",
        "msg_cfg_saved": "Config saved",
        "help_quickref_title": "Command Quick Reference",
        "help_global_opts": "Global Options",
        "help_examples": "Examples",
        "cmd_login": "Login and save cookies",
        "cmd_logout": "Logout",
        "cmd_list_servers": "List servers",
        "cmd_create_server": "Create server",
        "cmd_server_action": "Server action",
        "cmd_add_mod": "Add mod",
        "cmd_del_mod": "Delete mod",
        "cmd_toggle_mod": "Toggle mod enable/disable",
        "cmd_update_mod": "Update mod",
        "cmd_list_mods": "List mods",
        "cmd_rollback": "Rollback backup",
        "help_username": "Username",
        "help_password": "Password",
        "help_save_cookies": "Save cookies after login",
        "help_server_type": "Server type (e.g. Fabric / Vanilla)",
        "help_server_ver": "Server version (e.g. 1.20.1)",
        "help_display_name": "Display name",
        "help_server_id": "Server ID",
        "help_action": "Action (start / stop / backup / ls-backup / usage)",
        "help_mod_id": "Modrinth mod ID",
        "help_version_id": "Mod version ID (optional)",
        "help_no_auto_update": "Disable auto update",
        "help_file_name": "Backup file name",
        "title_login": "Login",
        "title_logout": "Logout",
        "title_list_servers": "Server List",
        "title_create_server": "Create Server",
        "title_server_action": "Server Action",
        "title_add_mod": "Add Mod",
        "title_del_mod": "Delete Mod",
        "title_toggle_mod": "Toggle Mod",
        "title_update_mod": "Update Mod",
        "title_list_mods": "Mod List",
        "title_rollback": "Rollback",
        "desc_login": "Login and fetch session cookies\nRequired: username, password",
        "desc_logout": "Logout (requires cookies)",
        "desc_list_servers": "List servers owned by user",
        "desc_create_server": "Create a new server",
        "desc_server_action": "Perform server action",
        "desc_add_mod": "Add mod to server",
        "desc_del_mod": "Remove mod from server",
        "desc_toggle_mod": "Toggle mod enable/disable",
        "desc_update_mod": "Update server mod",
        "desc_list_mods": "List server mods",
        "desc_rollback": "Rollback server backup",
    },
}


def hex_to_rgb(hex_color: str):
    hex_color = hex_color.lstrip("#")
    return int(hex_color[0:2], 16), int(hex_color[2:4], 16), int(hex_color[4:6], 16)


def colorize(text: str, hex_color: str, bold: bool = False):
    r, g, b = hex_to_rgb(hex_color)
    bold_code = "1;" if bold else ""
    return f"\x1b[{bold_code}38;2;{r};{g};{b}m{text}\x1b[0m"


def c(text: str, name: str, bold: bool = False):
    return colorize(text, PALETTE[name], bold=bold)


def header(title: str):
    print(c(f"\n== {title} ==", "mauve", bold=True))


def status_line(status: int):
    if status == 200:
        return c(f"{status} OK", "rosewater", bold=True)
    if status in (201, 202, 204):
        return c(f"{status} ACCEPTED", "flamingo", bold=True)
    if status in (400, 401, 403, 404):
        return c(f"{status} CLIENT ERROR", "maroon", bold=True)
    if status >= 500:
        return c(f"{status} SERVER ERROR", "maroon", bold=True)
    return c(str(status), "lavender", bold=True)


def print_response(status: int, response):
    print(c("Status:", "lavender", bold=True), status_line(status))
    data = None
    if response is not None:
        try:
            data = response.json()
        except Exception:
            data = None
    if data is None:
        print(c("Body:", "lavender", bold=True), c("<non-json response>", "fg_muted"))
        return
    pretty = json.dumps(data, indent=2, ensure_ascii=True)
    print(c("Body:", "lavender", bold=True))
    print(c(pretty, "fg_subtext0"))


def load_cookies_or_exit(term: wf.HttpTester, cookie_path: str):
    try:
        term.load_cookies(cookie_path)
    except FileNotFoundError:
        print(c(f"cookies file not found: {cookie_path}", "maroon", bold=True))
        sys.exit(1)


def set_base_url(base_url: str):
    if base_url:
        wf.BASE_URL = base_url.rstrip("/")


def cmd_login(args):
    set_base_url(args.base_url)
    header(t(args.lang, "title_login"))
    term = wf.HttpTester(verbose=False)
    status, response = term.login_and_verify(args.username, args.password)
    print_response(status, response)
    if args.save_cookies:
        term.save_cookies(args.cookies)
        print(c("Saved cookies to:", "lavender", bold=True), c(args.cookies, "fg_text"))


def cmd_logout(args):
    set_base_url(args.base_url)
    header(t(args.lang, "title_logout"))
    term = wf.HttpTester(verbose=False)
    load_cookies_or_exit(term, args.cookies)
    status, response = term.log_out()
    print_response(status, response)


def cmd_list_servers(args):
    set_base_url(args.base_url)
    header(t(args.lang, "title_list_servers"))
    term = wf.HttpTester(verbose=False)
    load_cookies_or_exit(term, args.cookies)
    status, response = term.list_servers()
    print_response(status, response)


def cmd_create_server(args):
    set_base_url(args.base_url)
    header(t(args.lang, "title_create_server"))
    term = wf.HttpTester(verbose=False)
    load_cookies_or_exit(term, args.cookies)
    status, response = term.create_server(args.server_type, args.server_ver, args.display_name)
    print_response(status, response)


def cmd_server_action(args):
    set_base_url(args.base_url)
    header(f"{t(args.lang, 'title_server_action')}: {args.action}")
    term = wf.HttpTester(verbose=False)
    load_cookies_or_exit(term, args.cookies)
    status, response = term.server(args.server_id, args.action)
    print_response(status, response)


def cmd_add_mod(args):
    set_base_url(args.base_url)
    header(t(args.lang, "title_add_mod"))
    term = wf.HttpTester(verbose=False)
    load_cookies_or_exit(term, args.cookies)
    status, response = term.add_mod_async(
        args.server_id,
        args.mod_id,
        ver_id=args.version_id,
        auto_update=not args.no_auto_update,
    )
    if status != 200 or response is None:
        print_response(status, response)
        return
    try:
        data = response.json()
    except Exception:
        print_response(status, response)
        return

    job_id = data.get("job_id")
    if not job_id:
        print_response(status, response)
        return

    stream = term.open_mod_job_stream(job_id)
    if stream is None:
        print_response(status, response)
        return
    if stream.status_code != 200:
        print_response(stream.status_code, stream)
        return

    state_lock = threading.Lock()
    state = {
        "stage": "queued",
        "percent": 0,
        "mod_id": args.mod_id,
        "mod_name": "",
        "message": "",
        "error": "",
        "done": False,
        "failed": False,
    }

    def stage_color(stage: str) -> str:
        stage = (stage or "").lower()
        if stage in ("queued", "resolving"):
            return "lavender"
        if stage in ("downloading",):
            return "flamingo"
        if stage in ("installing", "installed"):
            return "pink"
        if stage in ("skipped",):
            return "fg_muted"
        if stage in ("completed",):
            return "rosewater"
        if stage in ("failed",):
            return "maroon"
        return "fg_text"

    def render_line(spin_char: str) -> str:
        with state_lock:
            pct = state["percent"] if state["percent"] is not None else 0
            pct = max(0, min(int(pct), 100))
            bar_len = 30
            filled = int(pct / 100 * bar_len)
            bar = "#" * filled + "-" * (bar_len - filled)
            stage = state["stage"]
            mod_name = state["mod_name"] or state["mod_id"] or ""
            msg = state["message"] or ""
        stage_text = c(stage, stage_color(stage), bold=True)
        parts = [f"{spin_char} [{bar}] {pct:3d}% {stage_text}"]
        if mod_name:
            parts.append(mod_name)
        if msg:
            parts.append(msg)
        return " | ".join(parts)

    stop_flag = threading.Event()

    def spinner():
        chars = "|/-\\"
        idx = 0
        while not stop_flag.is_set():
            line = render_line(chars[idx % len(chars)])
            sys.stdout.write("\r" + line + " " * 10)
            sys.stdout.flush()
            idx += 1
            time.sleep(0.12)
        line = render_line(chars[idx % len(chars)])
        sys.stdout.write("\r" + line + " " * 10 + "\n")
        sys.stdout.flush()

    t_spin = threading.Thread(target=spinner, daemon=True)
    t_spin.start()

    def iter_sse_events(response):
        event = {}
        for raw in response.iter_lines(decode_unicode=True):
            if raw is None:
                continue
            line = raw.strip("\r")
            if not line:
                data = event.get("data")
                event = {}
                if data:
                    try:
                        yield json.loads(data)
                    except Exception:
                        continue
                continue
            if line.startswith(":"):
                continue
            if line.startswith("data:"):
                event["data"] = line[len("data:") :].strip()
            elif line.startswith("event:"):
                event["event"] = line[len("event:") :].strip()
            elif line.startswith("id:"):
                event["id"] = line[len("id:") :].strip()

    try:
        for ev in iter_sse_events(stream):
            with state_lock:
                state["stage"] = ev.get("stage", state["stage"])
                if ev.get("percent") is not None:
                    state["percent"] = ev.get("percent")
                if ev.get("mod_id"):
                    state["mod_id"] = ev.get("mod_id")
                if ev.get("mod_name"):
                    state["mod_name"] = ev.get("mod_name")
                if ev.get("message"):
                    state["message"] = ev.get("message")
                if ev.get("error"):
                    state["error"] = ev.get("error")
                if state["stage"] in ("completed", "failed"):
                    state["done"] = True
                    state["failed"] = state["stage"] == "failed"
                    break
    finally:
        stop_flag.set()
        try:
            stream.close()
        except Exception:
            pass

    if state.get("failed"):
        err_msg = state.get("error") or "unknown error"
        print(c("Error:", "maroon", bold=True), c(err_msg, "fg_text"))
    else:
        print(c("Done:", "lavender", bold=True), c("mod install completed", "fg_text"))


def cmd_del_mod(args):
    set_base_url(args.base_url)
    header(t(args.lang, "title_del_mod"))
    term = wf.HttpTester(verbose=False)
    load_cookies_or_exit(term, args.cookies)
    status, response = term.del_mod(args.server_id, args.mod_id)
    print_response(status, response)


def cmd_toggle_mod(args):
    set_base_url(args.base_url)
    header(t(args.lang, "title_toggle_mod"))
    term = wf.HttpTester(verbose=False)
    load_cookies_or_exit(term, args.cookies)
    status, response = term.toggle_mod(args.server_id, args.mod_id)
    print_response(status, response)


def cmd_update_mod(args):
    set_base_url(args.base_url)
    header(t(args.lang, "title_update_mod"))
    term = wf.HttpTester(verbose=False)
    load_cookies_or_exit(term, args.cookies)
    status, response = term.update_mod(args.server_id, args.mod_id)
    print_response(status, response)


def cmd_list_mods(args):
    set_base_url(args.base_url)
    header(t(args.lang, "title_list_mods"))
    term = wf.HttpTester(verbose=False)
    load_cookies_or_exit(term, args.cookies)
    status, response = term.list_mods(args.server_id)
    print_response(status, response)


def cmd_rollback(args):
    set_base_url(args.base_url)
    header(t(args.lang, "title_rollback"))
    term = wf.HttpTester(verbose=False)
    load_cookies_or_exit(term, args.cookies)
    status, response = term.server_rollback(args.file_name, args.server_id)
    print_response(status, response)


def detect_lang(argv, cfg_lang: str = ""):
    for idx, arg in enumerate(argv):
        if arg in ("--lang", "-L") and idx + 1 < len(argv):
            val = argv[idx + 1].lower()
            if val in ("zh", "en"):
                return val
        if arg.startswith("--lang="):
            val = arg.split("=", 1)[1].lower()
            if val in ("zh", "en"):
                return val
    env_lang = os.getenv("WF_LANG", "").lower()
    if env_lang in ("zh", "en"):
        return env_lang
    cfg_lang = (cfg_lang or "").lower()
    if cfg_lang in ("zh", "en"):
        return cfg_lang
    return "zh"


def resolve_config_path() -> str:
    env_path = os.getenv("WF_CONFIG", "").strip()
    if env_path:
        return env_path
    home = os.path.expanduser("~")
    return os.path.join(home, ".wf_cli_config.json")


def load_config(path: str):
    try:
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f), True
    except FileNotFoundError:
        return {}, False
    except Exception:
        return {}, False


def save_config(path: str, data: dict):
    directory = os.path.dirname(path)
    if directory and not os.path.exists(directory):
        os.makedirs(directory, exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=True, indent=2)


def t(lang: str, key: str) -> str:
    return TRANSLATIONS.get(lang, TRANSLATIONS["zh"]).get(key, key)


def help_text(lang: str, key: str, color: str = "fg_subtext0") -> str:
    return c(t(lang, key), color)


def command_reference(
    lang: str, default_base_url: str, default_cookies: str, cfg_path: str
) -> str:
    lines = []
    divider = c("─" * 72, "fg_muted")
    lines.append(divider)
    lines.append(c(t(lang, "help_quickref_title"), "mauve", bold=True))
    lines.append(divider)
    lines.append(
        c(t(lang, "help_global_opts"), "lavender", bold=True)
        + " "
        + c(
            f"--lang {{zh,en}}  --base-url <url>  --cookies <path>  --cfg\n"
            f"    (default base-url: {default_base_url}, cookies: {default_cookies})",
            "fg_subtext0",
        )
    )
    lines.append(c(f"{t(lang, 'help_config_path')}: {cfg_path}", "fg_muted"))
    lines.append(divider)
    lines.append(c(t(lang, "cmd_login"), "pink", bold=True))
    lines.append(
        c("  login -u <username> -p <password> [--save-cookies]", "fg_subtext0")
    )
    lines.append(c(t(lang, "cmd_logout"), "pink", bold=True))
    lines.append(c("  logout", "fg_subtext0"))
    lines.append(c(t(lang, "cmd_list_servers"), "pink", bold=True))
    lines.append(c("  list-servers", "fg_subtext0"))
    lines.append(c(t(lang, "cmd_create_server"), "pink", bold=True))
    lines.append(
        c(
            "  create-server --server-type <type> --server-ver <ver> --display-name <name>",
            "fg_subtext0",
        )
    )
    lines.append(c(t(lang, "cmd_server_action"), "pink", bold=True))
    lines.append(
        c(
            "  server <server_id> <action>  (action: start|stop|backup|ls-backup|usage)",
            "fg_subtext0",
        )
    )
    lines.append(c(t(lang, "cmd_add_mod"), "pink", bold=True))
    lines.append(
        c(
            "  add-mod <server_id> <mod_id> [--version-id <id>] [--no-auto-update]",
            "fg_subtext0",
        )
    )
    lines.append(c(t(lang, "cmd_del_mod"), "pink", bold=True))
    lines.append(c("  del-mod <server_id> <mod_id>", "fg_subtext0"))
    lines.append(c(t(lang, "cmd_toggle_mod"), "pink", bold=True))
    lines.append(c("  toggle-mod <server_id> <mod_id>", "fg_subtext0"))
    lines.append(c(t(lang, "cmd_update_mod"), "pink", bold=True))
    lines.append(c("  update-mod <server_id> <mod_id>", "fg_subtext0"))
    lines.append(c(t(lang, "cmd_list_mods"), "pink", bold=True))
    lines.append(c("  list-mods <server_id>", "fg_subtext0"))
    lines.append(c(t(lang, "cmd_rollback"), "pink", bold=True))
    lines.append(c("  rollback <server_id> <file_name>", "fg_subtext0"))

    lines.append(divider)
    lines.append(c(t(lang, "help_examples"), "lavender", bold=True))
    if lang == "zh":
        lines.append(
            c(
                "  python work_flow_cli.py -L zh login -u car -p 123",
                "fg_subtext0",
            )
        )
        lines.append(
            c(
                "  python work_flow_cli.py --base-url http://localhost:8080 list-servers",
                "fg_subtext0",
            )
        )
        lines.append(
            c(
                "  python work_flow_cli.py --cfg --lang zh --base-url http://localhost:8080",
                "fg_subtext0",
            )
        )
    else:
        lines.append(
            c(
                "  python work_flow_cli.py -L en login -u car -p 123",
                "fg_subtext0",
            )
        )
        lines.append(
            c(
                "  python work_flow_cli.py --base-url http://localhost:8080 list-servers",
                "fg_subtext0",
            )
        )
        lines.append(
            c(
                "  python work_flow_cli.py --cfg --lang en --base-url http://localhost:8080",
                "fg_subtext0",
            )
        )

    lines.append(divider)
    return "\n".join(lines)


class FancyFormatter(
    argparse.ArgumentDefaultsHelpFormatter, argparse.RawTextHelpFormatter
):
    pass


def build_parser(
    lang: str, default_base_url: str, default_cookies: str, cfg_path: str
):
    parser = argparse.ArgumentParser(
        description=c(t(lang, "cli_desc"), "fg_text", bold=True),
        formatter_class=FancyFormatter,
    )
    parser.add_argument(
        "-L",
        "--lang",
        choices=["zh", "en"],
        default=lang,
        help=help_text(lang, "help_lang", "lavender"),
    )
    parser.add_argument(
        "--base-url",
        default=default_base_url,
        help=help_text(lang, "help_base_url", "fg_subtext0"),
    )
    parser.add_argument(
        "--cookies",
        default=default_cookies,
        help=help_text(lang, "help_cookies", "fg_subtext0"),
    )
    parser.add_argument(
        "--cfg",
        action="store_true",
        help=help_text(lang, "help_cfg", "lavender"),
    )
    parser.epilog = (
        c(t(lang, "help_default_env"), "fg_muted")
        + "\n\n"
        + command_reference(lang, default_base_url, default_cookies, cfg_path)
    )

    sub = parser.add_subparsers(dest="command")

    login = sub.add_parser(
        "login",
        help=help_text(lang, "cmd_login", "mauve"),
        description=help_text(lang, "desc_login", "fg_text"),
        formatter_class=FancyFormatter,
    )
    login.add_argument("-u", "--username", required=True, help=help_text(lang, "help_username"))
    login.add_argument("-p", "--password", required=True, help=help_text(lang, "help_password"))
    login.add_argument(
        "--save-cookies",
        action="store_true",
        default=True,
        help=help_text(lang, "help_save_cookies"),
    )
    login.set_defaults(func=cmd_login)

    logout = sub.add_parser(
        "logout",
        help=help_text(lang, "cmd_logout", "mauve"),
        description=help_text(lang, "desc_logout", "fg_text"),
        formatter_class=FancyFormatter,
    )
    logout.set_defaults(func=cmd_logout)

    ls_srv = sub.add_parser(
        "list-servers",
        help=help_text(lang, "cmd_list_servers", "mauve"),
        description=help_text(lang, "desc_list_servers", "fg_text"),
        formatter_class=FancyFormatter,
    )
    ls_srv.set_defaults(func=cmd_list_servers)

    cs = sub.add_parser(
        "create-server",
        help=help_text(lang, "cmd_create_server", "mauve"),
        description=help_text(lang, "desc_create_server", "fg_text"),
        formatter_class=FancyFormatter,
    )
    cs.add_argument("--server-type", required=True, help=help_text(lang, "help_server_type"))
    cs.add_argument("--server-ver", required=True, help=help_text(lang, "help_server_ver"))
    cs.add_argument("--display-name", required=True, help=help_text(lang, "help_display_name"))
    cs.set_defaults(func=cmd_create_server)

    srv = sub.add_parser(
        "server",
        help=help_text(lang, "cmd_server_action", "mauve"),
        description=help_text(lang, "desc_server_action", "fg_text"),
        formatter_class=FancyFormatter,
    )
    srv.add_argument("server_id", help=help_text(lang, "help_server_id"))
    srv.add_argument(
        "action",
        choices=["start", "stop", "backup", "ls-backup", "usage", "details"],
        help=help_text(lang, "help_action"),
    )
    srv.set_defaults(func=cmd_server_action)

    add_mod = sub.add_parser(
        "add-mod",
        help=help_text(lang, "cmd_add_mod", "mauve"),
        description=help_text(lang, "desc_add_mod", "fg_text"),
        formatter_class=FancyFormatter,
    )
    add_mod.add_argument("server_id", help=help_text(lang, "help_server_id"))
    add_mod.add_argument("mod_id", help=help_text(lang, "help_mod_id"))
    add_mod.add_argument("--version-id", default="", help=help_text(lang, "help_version_id"))
    add_mod.add_argument(
        "--no-auto-update",
        action="store_true",
        help=help_text(lang, "help_no_auto_update"),
    )
    add_mod.set_defaults(func=cmd_add_mod)

    del_mod = sub.add_parser(
        "del-mod",
        help=help_text(lang, "cmd_del_mod", "mauve"),
        description=help_text(lang, "desc_del_mod", "fg_text"),
        formatter_class=FancyFormatter,
    )
    del_mod.add_argument("server_id", help=help_text(lang, "help_server_id"))
    del_mod.add_argument("mod_id", help=help_text(lang, "help_mod_id"))
    del_mod.set_defaults(func=cmd_del_mod)

    toggle_mod = sub.add_parser(
        "toggle-mod",
        help=help_text(lang, "cmd_toggle_mod", "mauve"),
        description=help_text(lang, "desc_toggle_mod", "fg_text"),
        formatter_class=FancyFormatter,
    )
    toggle_mod.add_argument("server_id", help=help_text(lang, "help_server_id"))
    toggle_mod.add_argument("mod_id", help=help_text(lang, "help_mod_id"))
    toggle_mod.set_defaults(func=cmd_toggle_mod)

    update_mod = sub.add_parser(
        "update-mod",
        help=help_text(lang, "cmd_update_mod", "mauve"),
        description=help_text(lang, "desc_update_mod", "fg_text"),
        formatter_class=FancyFormatter,
    )
    update_mod.add_argument("server_id", help=help_text(lang, "help_server_id"))
    update_mod.add_argument("mod_id", help=help_text(lang, "help_mod_id"))
    update_mod.set_defaults(func=cmd_update_mod)

    list_mods = sub.add_parser(
        "list-mods",
        help=help_text(lang, "cmd_list_mods", "mauve"),
        description=help_text(lang, "desc_list_mods", "fg_text"),
        formatter_class=FancyFormatter,
    )
    list_mods.add_argument("server_id", help=help_text(lang, "help_server_id"))
    list_mods.set_defaults(func=cmd_list_mods)

    rollback = sub.add_parser(
        "rollback",
        help=help_text(lang, "cmd_rollback", "mauve"),
        description=help_text(lang, "desc_rollback", "fg_text"),
        formatter_class=FancyFormatter,
    )
    rollback.add_argument("server_id", help=help_text(lang, "help_server_id"))
    rollback.add_argument("file_name", help=help_text(lang, "help_file_name"))
    rollback.set_defaults(func=cmd_rollback)

    return parser


def main():
    cfg_path = resolve_config_path()
    cfg_data, _ = load_config(cfg_path)
    lang = detect_lang(sys.argv, cfg_data.get("lang", ""))
    default_base_url = os.getenv(
        "WF_BASE_URL", cfg_data.get("base_url", wf.BASE_URL)
    )
    default_cookies = os.getenv("WF_COOKIES", cfg_data.get("cookies", "cookies.json"))
    parser = build_parser(lang, default_base_url, default_cookies, cfg_path)
    args = parser.parse_args()

    try:
        if getattr(args, "command", None) is None:
            if getattr(args, "cfg", False):
                save_config(
                    cfg_path,
                    {
                        "lang": args.lang,
                        "base_url": args.base_url,
                        "cookies": args.cookies,
                    },
                )
                print(
                    c(t(args.lang, "msg_cfg_saved") + ":", "lavender", bold=True),
                    c(cfg_path, "fg_text"),
                )
                return
            parser.print_help()
            return

        if getattr(args, "cfg", False):
            save_config(
                cfg_path,
                {
                    "lang": args.lang,
                    "base_url": args.base_url,
                    "cookies": args.cookies,
                },
            )
            print(
                c(t(args.lang, "msg_cfg_saved") + ":", "lavender", bold=True),
                c(cfg_path, "fg_text"),
            )
        args.func(args)
    except Exception as exc:
        print(c("Error:", "maroon", bold=True), c(str(exc), "fg_text"))
        sys.exit(1)


if __name__ == "__main__":
    main()
