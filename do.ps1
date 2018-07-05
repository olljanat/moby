#docker build -t nativebuildimage -f Dockerfile.windows -m 2GB .
$DOCKER_GITCOMMIT=(git rev-parse --short HEAD)
docker run --name binaries -e DOCKER_GITCOMMIT=$DOCKER_GITCOMMIT -m 2GB nativebuildimage hack\make.ps1 -Binary
docker cp binaries:C:\go\src\github.com\docker\docker\bundles\docker.exe C:\HostPath\docker.exe
docker cp binaries:C:\go\src\github.com\docker\docker\bundles\dockerd.exe C:\HostPath\dockerd.exe
docker rm binaries
