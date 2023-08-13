#!/bin/bash 
clear


# compile go program first
go build -o cmd/gobc/main cmd/gobc/main.go
for (( i=0; i<=$1; i++))
do
    echo "####################${i}######################"

    cmd/gobc/main $i $i
    if [ $? -ne 0 ]; then
        echo "go run command failed with exit code $?"
        exit 1
    fi

    DEBUG=1 ./.venv/bin/python tests/main.py
    if [ $? -ne 0 ]; then
        echo "python command failed with exit code $?"
        exit 1
    fi

    echo "*----------------------------------------------*"

    diff -s registers-test.json registers-validate.json
    if [ $? -ne 0 ]; then
        echo "diff command failed with exit code $?"
        exit 1
    fi
    echo "############################################"

done

echo "All tests passed!!!"
