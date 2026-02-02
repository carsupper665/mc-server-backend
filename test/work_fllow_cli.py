import argparse

if __name__ == "__main__":
    parser = argparse.ArgumentParser()

    loging_arg = parser.add_subparsers(dest="login", description="login workflow")
    loging_arg.add_parser("username", type=str)
    loging_arg.add_parser("email", type=str)
    loging_arg.add_parser("password", type=str, require=True)

    server = parser.add_subparsers(dest="server", description="add server")
    server.add_parser("list", type=str)
    server.add_parser("add", type=str)
    server.add_parser("delete", type=str)


