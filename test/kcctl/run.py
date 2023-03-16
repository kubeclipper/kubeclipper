import pytest
from config.config import CMD_CONFIG

if __name__ == '__main__':
    if CMD_CONFIG.errlog:
        with open("err.log","w") as f:
            f.write("ERROR LOG\n\n\n")
    pytest.main(['--html=report.html', '--self-contained-html', '-s'])