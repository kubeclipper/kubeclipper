import subprocess
from config.config import CMD_CONFIG


class CMD:

    def __init__(self, stdout=CMD_CONFIG.stdout, stderr=CMD_CONFIG.stderr, output=CMD_CONFIG.output, errlog=CMD_CONFIG.errlog):
        self.stdout = stdout
        self.stderr = stderr
        self.output = output
        self.errlog = errlog

    def run(self, command):
        if self.output:
            print(command)
        command_list = command.split()
        p =subprocess.Popen(command_list, shell=False, stdout=self.stdout, stderr=self.stderr)
        stdout, stderr = p.communicate()
        stdout = stdout.decode("utf-8")
        stderr = stderr.decode("utf-8")
        if self.errlog:
            with open("err.log","a") as f:
                f.write("{}\n\n".format(stderr))
        return stdout, stderr