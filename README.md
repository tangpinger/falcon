# falcon
A quant trading tool designed for crypto-market

# local test
0. setup ss and enable https proxy at port 1087
1. setup local proxy
```
$ export http_proxy=http://127.0.0.1:1087;export https_proxy=http://127.0.0.1:1087;
```

2. input api keys in config file and run pixiu
```
$ cd venus/cmd/plutus
$ go build
$ ./plutus pixiu --config=../../../config/demo.toml --log_output_level="default:debug"
```