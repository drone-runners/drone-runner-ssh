# drone-runner-ssh

The ssh runner executes pipelines on a remote server using the ssh protocol. This runner is intended for workloads that are not suitable for running inside containers. Posix and Windows workloads supported. Drone server 1.2.1 or higher is required.

## Community and Support

[Harness Community Slack](https://join.slack.com/t/harnesscommunity/shared_invite/zt-y4hdqh7p-RVuEQyIl5Hcx4Ck8VCvzBw) - Join the #drone slack channel to connect with our engineers and other users running Drone CI.

[Harness Community Forum](https://community.harness.io/) - Ask questions, find answers, and help other users.

[Report A Bug](https://community.harness.io/c/bugs/17) - Find a bug? Please report in our forum under Drone Bugs. Please provide screenshots and steps to reproduce.

[Events](https://www.meetup.com/harness/) - Keep up to date with Drone events and check out previous events [here](https://www.youtube.com/watch?v=Oq34ImUGcHA&list=PLXsYHFsLmqf3zwelQDAKoVNmLeqcVsD9o).

## Developer resources

### Testing locally

Running an ssh server locally is easy.

``` bash
docker run -it --rm \      
  --name=openssh-server \ 
  --hostname=openssh  \
  -e PUID=1000 \
  -e PGID=1000 \
  -e PASSWORD_ACCESS=true \
  -e USER_PASSWORD=secret  \
  -e USER_NAME=user \
  -p 2222:2222 \
  lscr.io/linuxserver/openssh-server:latest
```

You can then reference the server in your pipeline.

``` yaml
kind: pipeline
type: ssh
name: default

server:
  host: 0.0.0.0:2222
  user: user
  password: secret

clone:
  disable: true

steps:
  - name: test
    commands:
      - echo "hello world"
```
