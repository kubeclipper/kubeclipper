from common.cmd import CMD
from config.config import KCCTL_CONFIG


def test_login():
    cmd = CMD()
    cmd.run("kcctl login --host http://{} --username admin --password Thinkbig1".format(KCCTL_CONFIG.server))
