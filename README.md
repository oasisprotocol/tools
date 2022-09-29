# tools
Various useful tools and scripts


## System update
```
sudo apt update && sudo apt upgrade -y
```

## Library installation
```
sudo apt install make clang pkg-config libssl-dev build-essential git ncdu bsdmainutils -y < "/dev/null"
```
You can also add other libraries here. For example; jq, screen, curl etc.

## Go setup
```
cd $HOME
wget -O go1.18.2.linux-amd64.tar.gz https://go.dev/dl/go1.18.2.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz && rm go1.18.2.linux-amd64.tar.gz
echo 'export GOROOT=/usr/local/go' >> $HOME/.bashrc
echo 'export GOPATH=$HOME/go' >> $HOME/.bashrc
echo 'export GO111MODULE=on' >> $HOME/.bashrc
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> $HOME/.bashrc && . $HOME/.bashrc
```
You can see your version by 'go version' command.

## Docker installation
```
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
```

## Docker compose installation
```
curl -SL https://github.com/docker/compose/releases/download/v2.5.0/docker-compose-linux-x86_64 -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
sudo ln -s /usr/local/bin/docker-compose /usr/bin/docker-compose
```

## Docker initialization
```
docker-compose down -v
docker-compose up -d
```

## Rust installation. You can install by default by following option 1
```
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
source $HOME/.cargo/env
```
you can see your rust version by 'rustc --version' command




