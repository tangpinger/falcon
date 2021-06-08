#!/bin/bash
export http_proxy=http://127.0.0.1:1087;export https_proxy=http://127.0.0.1:1087;


./plutus pixiu --config=../../../config/pixiu.toml --log_output_level="exchange:debug,fetcher:debug,oracle:debug,trader:debug"
