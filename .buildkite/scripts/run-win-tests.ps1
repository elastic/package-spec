$ErrorActionPreference = "Stop" # set -e
# Forcing to checkout again all the files with a correct autocrlf.
# Doing this here because we cannot set git clone options before.
function fixCRLF {
    Write-Host "--- Fixing CRLF in git checkout"
    git config core.autocrlf input
    git rm --quiet --cached -r .
    git reset --quiet --hard
}
function withGolang($version) {
    Write-Host "--- Install golang $version"
    choco install -y golang --version $version
    $env:ChocolateyInstall = Convert-Path "$((Get-Command choco).Path)\..\.."
    Import-Module "$env:ChocolateyInstall\helpers\chocolateyProfile.psm1"
    refreshenv
    go version
    go env
}
function installGoDependencies {
    $installPackages = @(
        "github.com/magefile/mage"
    )
    foreach ($pkg in $installPackages) {
        go install "$pkg@$env:SETUP_MAGE_VERSION"
    }
}

fixCRLF
withGolang $env:GO_VERSION
installGoDependencies

echo "--- Downloading Go modules"
go mod download -x

echo "--- Running unit tests"
$ErrorActionPreference = "Continue" # set +e

go run gotest.tools/gotestsum --format testname --junitfile junit-win-report.xml -- -v ./code/go/...

$EXITCODE=$LASTEXITCODE
$ErrorActionPreference = "Stop"

Exit $EXITCODE
