$VERSION = "v19.03.5-olljanat2"

cd C:\gopath\src\github.com\docker\docker
docker build -t nativebuildimage -f Dockerfile.windows .
$DOCKER_GITCOMMIT=(git rev-parse --short HEAD)
docker run --name binaries -e DOCKER_GITCOMMIT=$DOCKER_GITCOMMIT -e VERSION=$VERSION nativebuildimage hack\make.ps1 -Binary
docker cp binaries:C:\go\src\github.com\docker\docker\bundles\dockerd.exe C:\HostPath\dockerd.exe
docker rm binaries

Write-Host "Build time $(Get-Date -Format u)"