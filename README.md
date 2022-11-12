
# API-Failover  
Using the Cloudflare API to change DNS records to maximize uptime
## Badges  
[![Continuous Integration](https://github.com/slashtechno/api-failover/actions/workflows/ci.yml/badge.svg)](https://github.com/slashtechno/api-failover/actions/workflows/ci.yml) [![MIT License](https://img.shields.io/badge/License-MIT-green.svg)](https://choosealicense.com/licenses/mit/) [![Create and publish a Docker image](https://github.com/slashtechno/api-failover/actions/workflows/build-Docker-image.yml/badge.svg)](https://github.com/slashtechno/api-failover/actions/workflows/build-Docker-image.yml)  
## Usage/Examples  
Necessary data can be passed via environment vairables or CLI flags  
```bash
api-failover --primary 0.0.0.0,0.0.0.1,0.0.0.2 --backup 0.0.0.3,0.0.0.4,0.0.0.5 --cloudflareapitoken token --cloudflarezoneid CLOUDFLAREZONEID --recordname RECORDNAME
# Is equivalent to 
CLOUDFLARE_API_TOKEN="token" CLOUDFLARE_ZONE_ID="zoneid" RECORD_NAME="RECORDNAME" PRIMARY_IPs="0.0.0.0,0.0.0.1,0.0.0.2" BACKUP_IPs="0.0.0.3,0.0.0.4,0.0.0.5" api-failover
# and
docker run -e CLOUDFLARE_API_TOKEN="token" -e CLOUDFLARE_ZONE_ID="zoneid" -e RECORD_NAME="RECORDNAME" -e PRIMARY_IPs="0.0.0.0,0.0.0.1,0.0.0.2" -e BACKUP_IPs="0.0.0.3,0.0.0.4,0.0.0.5" -it --rm ghcr.io/slashtechno/api-failover
```  
For full flag usage, run:   
`api-failover --help`  
### Pinging on Linux  
In some cases, an error may be thrown when the program attempts to ping the specified hosts. The simplest way to alleviate this is to run the program as root or in Docker.  
For more information, check the Linux section in the  [pro-bing](https://github.com/prometheus-community/pro-bing#linux) README
## Installation  
### Precompiled releases   
Precompiled releases are build automatically by Github Actions and can be downloaded from the [releases](https://github.com/slashtechno/api-failover/releases) page  
After downloading for the appropriate platform, the program can be run directly  
### Docker  
Docker images can either be built locally, or pulled from the [Github Container Registry](https://github.com/slashtechno/api-failover/pkgs/container/api-failover)  
An advantage to running with Docker is that the software is isolated which can reduce the possiblity of errors. In addition, it can increase security.
To pull and run, the following commands can be used:
```bash
docker pull ghcr.io/slashtechno/api-failover:latest 
# If the image isn't pulled manually, the following command will pull it automatically before running
docker run -e CLOUDFLARE_API_TOKEN="token" -e CLOUDFLARE_ZONE_ID="zoneid" -e RECORD_NAME="RECORDNAME" -e PRIMARY_IPs="0.0.0.0,0.0.0.1,0.0.0.2" -e BACKUP_IPs="0.0.0.3,0.0.0.4,0.0.0.5" -it --rm ghcr.io/slashtechno/api-failover
```  
### Compiling locally  
In order to compile locally, Go must be installed  
```bash
git clone https://github.com/slashtechno/api-failover/
cd api-failover
go install
```
## Roadmap  
- [ ] Add support for CNAME records  
