#!/bin/bash 
#
# This bash script is designed to compile and test a Go program, specifically for a range of opcode values. 
# The script first removes any existing compiled Go binary and then compiles the Go program. It then checks 
# if the user has requested help, and if so, it displays a help message and exits.
#
# If the user has not requested help, the script checks if the user has provided at least two arguments. 
# If not, it displays an error message and exits. If the user has provided two arguments, the script sets 
# the third argument to 1 if it was not provided.
# The script then prints the current date and time, and enters a nested loop. The outer loop runs for the 
# number of tests specified by the user, and the inner loop runs for each opcode in the range specified by 
# the user.
# 
# For each test, the script generates a random value and runs the Go program with the opcode and random value 
# as arguments. If the Go program exits with a non-zero status code, the script displays an error message and 
# exits.
# 
# The script then runs a Python script in debug mode. If the Python script exits with a non-zero status code, 
# the script displays an error message and exits.
# 
# The script then compares two JSON files using the diff command. If the diff command exits with a non-zero 
# status code, the script displays an error message and exits.
# 
# If all tests pass, the script displays a success message.
# Example Usage:
# To run this script with opcode range from 0 to 10 and run 1 test for each opcode, you would use the following command:
# 
#```bash
#./script.sh 0 10 1
#```
############################################################

# compile go program first
rm -f cmd/gobc/main
go build -o cmd/gobc/main cmd/gobc/main.go

# CLI help section
if [ "$1" == "-h" ] || [ "$1" == "--help" ]; then
  echo "Usage: `basename $0` [opcode_start] [opcode_end] [number_of_tests] [argvalue]"
  echo ""
  echo "[opcode_start]: The starting opcode value in integer"
  echo "[opcode_end]  : The ending opcode value in integer"
  echo "[number_of_tests]  : The number of tests to run for each opcode range"
  echo "[--argvalue]  : The value to replace the randomly generated number, parsed as a hexstring"
  echo ""
  echo "Description"
  echo "This script will run the tests for the given opcode range"
  echo "Using PyBoy code as validation that opcodes operate as expected"
  echo "All calls to use Motherboard to write to registers are mocked and "
  echo "all reads will return only `0xDA`"
  echo ""
  echo "Example: `basename $0` 0 10 1 --argvalue 0x1A"
  echo "This will run 1 test for each opcode from 0 to 10, with the argvalue replacing the random number"
  exit 0
fi

# Check if the required arguments are provided
if [ -z "$1" ] || [ -z "$2" ]; then
  echo "Error: You must provide at least two arguments."
  echo "Use `basename $0` -h for help."
  exit 1
fi

if [ -z "$3" ]
then
    arg3=1
else
    arg3=$3
fi

argvalue=$4

start=$(printf '%d' $1)
end=$(printf '%d' $2)

echo ""
date +%Y-%m-%d:%H:%M:%S
echo ""

for (( j=$start; j<=$end; j++))
do
    echo "*******************-- $j [0x$(printf '%X' $j)]--*************************"
    for (( i=0; i<$arg3; i++))
    do
        echo "Test        : $i"
        if [ -z "$argvalue" ]
        then
            randValue=$((256 + RANDOM % (65535 - 256 + 1)))
        else
            randValue=$(printf '%d' $argvalue)
        fi
        opcode=$(printf '%4X' $j)
        echo "Value       : $randValue [0x$(printf '%X' $randValue)]"
        echo "OpCode      : $opcode"
    
        cmd/gobc/main $opcode $randValue
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
        echo ""
    done
done

echo "All tests passed!!!"