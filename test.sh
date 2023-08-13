#!/bin/bash 
clear

# compile go program first
go build -o cmd/gobc/main cmd/gobc/main.go

if [ -z "$3" ]
then
    arg3=1
else
    arg3=$3
fi

for (( i=0; i<$arg3; i++))
do
    for (( j=$1; j<=$2; j++))
    do
        echo "####################${j}######################"

        cmd/gobc/main $j 255
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
done

echo "All tests passed!!!"