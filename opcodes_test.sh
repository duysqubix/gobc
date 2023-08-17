#!/bin/bash 
#
# Opcode Verification Script
#
# This script provides the user with the versatility to examine all opcodes, a specific range, an 
# individual opcode, and/or the capability to test each opcode n times.
# 
# The purpose of this script is to verify the correctness and functionality of the emulated LR35902 
# chipset, ensuring it performs as intended. The benchmark and validation it tests against is the logic 
# found in PyBoy, (https://github.com/Baekalfen/PyBoy).
# 
# The aforementioned is a functional GB & CGB Emulator written in Python. The script intentionally uses 
# random values that are passed to both gobc and pyboy for output testing.
# 
# Both programs output to the console and create two distinct json files that are compared using 'diff'. 
# The data dumped into the json files are the registries from both GoBC and PyBoy.
############################################################

function is_in_array() {
  local search_term="$1"
  shift
  local array=("$@")

  for item in "${array[@]}"; do
    if [[ $item == $search_term ]]; then
      echo "True"
      return 0
    fi
  done

  echo "False"
  return 1
}

ROOT=`git rev-parse --show-toplevel`
TEST_DIR=$ROOT/tests/pyboy
BIN_DIR=$ROOT/bin


GOBCBIN=$BIN_DIR/optest
PYBIN=`which python3`

rm -f $GOBCBIN

go build -o $GOBCBIN $ROOT/cmd/optest/main.go

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
if [ -z "$1" ]; then
  echo "Error: You must provide at least one argument."
  echo "Use `basename $0` -h for help."
  exit 1
fi

start=$(printf '%d' $1)

if [ -z "$2" ]; then 
    echo "Second argument not given, defaulting range to ${1}-${1}"
    end=$(printf '%d' $1)
else
    end=$(printf '%d' $2)
fi

if [ -z "$3" ]
then
    arg3=1
else
    arg3=$3
fi

argvalue=$4

echo ""
date +%Y-%m-%d:%H:%M:%S
echo ""

ILLEGAL_OPCODES=(D3 DB DD E3 E4 EB EC ED F4 FC FD)
for (( j=$start; j<=$end; j++))
do
    opcode_hex=$(printf '%02X' $j_str)

    if [ $(is_in_array $opcode_hex "${ILLEGAL_OPCODES[@]}") == "True" ]; then
        echo "*==============================================*"
        echo "| Skipping illegal opcode 0x$opcode_hex "
        echo "*==============================================*"
        continue
    fi

    if [ $j -gt 255 ]; then 
        echo "CB Prefix Command"

        # substract j by 255 to get the correct opcode
        j_str=$(($j-255))
        echo $j_str
        opcode_hex=$(printf 'CB %02X' $j_str)
    else
        opcode_hex=$(printf '%02X' $j_str)
    fi 

    echo "*******************-- $j [$opcode_hex]--*************************"
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
    
        $GOBCBIN $opcode $randValue
        if [ $? -ne 0 ]; then
            echo "go run command failed with exit code $?"
            exit 1
        fi

        DEBUG=1 $PYBIN $TEST_DIR/main.py
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