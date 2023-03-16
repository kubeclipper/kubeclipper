import subprocess


class CMD_CONFIG:

    stdout = subprocess.PIPE
    stderr = subprocess.PIPE
    output = False
    errlog = True


class KCCTL_CONFIG:

    server = '172.20.139.223'
    