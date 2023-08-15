from opcodes import *
from unittest.mock import Mock

import opcodes
import os 
import json 
import pandas as pd
import random

# EXPLICIT = ('INC_03', 'INC_04')


def get_os_env(key, default=None):
    try:
        return os.environ[key]
    except:
        return default
    
debug_on = True if get_os_env("DEBUG")  else False

# FUNC_ITR = EXPLICIT if debug_on else dir(opcodes)

class MB:
    def __init__(self, mock_return) -> None:
        self.mock_return = mock_return
        self.cgb = False
    
    def setitem(self, addr, value):
        # this is just so it doesn't complain..
        value &= 0xFF
        assert 0 <= value < 0x100, "Memory write error! Can't write %s to %s" % (hex(value), hex(addr))

        print(f"Writing 0x{value:X} to 0x{addr:X}")
        
    def getitem(self, addr):
        print(f"Reading from 0x{addr:X}")
        return self.mock_return
        

class DummyCPU:
    def set_bc(self, x):
        self.B = x >> 8
        self.C = x & 0x00FF

    def set_de(self, x):
        self.D = x >> 8
        self.E = x & 0x00FF

    def f_c(self):
        return (self.F & (1 << FLAGC)) != 0

    def f_h(self):
        return (self.F & (1 << FLAGH)) != 0

    def f_n(self):
        return (self.F & (1 << FLAGN)) != 0

    def f_z(self):
        return (self.F & (1 << FLAGZ)) != 0

    def f_nc(self):
        return (self.F & (1 << FLAGC)) == 0

    def f_nz(self):
        return (self.F & (1 << FLAGZ)) == 0

    def __init__(self, func_name=None):
        with open("registers-start.json", "r") as f:
            data = json.load(f)
        args = data.get('args', None) 
        if args:
            self.args = int(args)
            
        self.Name = data["name"]
        self.A =  int(data["a"]) & 0xFF
        self.F = int(data["f"]) & 0xFF
        self.B =  int(data["b"]) & 0xFF
        self.C =  int(data["c"]) & 0xFF
        self.D =  int(data["d"]) & 0xFF
        self.E =  int(data["e"]) & 0xFF
        self.HL =  int(data["hl"]) & 0xFFFF
        self.SP =  int(data["sp"]) & 0xFFFF
        self.PC = int(data["pc"]) & 0xFFFF
        self.func_name = f"{self.Name:X}"

        self.interrupts_flag_register = 0
        self.interrupts_enabled_register = 0
        self.interrupt_master_enable = False
        self.interrupt_queued = False

        self.mb = MB(mock_return=0xDA)

        self.halted = False
        self.stopped = False
        self.is_stuck = False
            
    def report(self):
        self.A &= 0xFF
        self.B &= 0xFF
        self.C &= 0xFF
        self.D &= 0xFF
        self.E &= 0xFF

        self.HL &= 0xFFFF
        self.SP &= 0xFFFF
        self.PC &= 0xFFFF
        
        return (
            f"PyBoy -- Post Instruction [{self.func_name}]\n" +
            f"A: {self.A:02X}({self.A}) F: {self.F:02X}({self.F}) <{(self.F>>4):04b}|ZNHC>\n" +
            f"B: {self.B:02X}({self.B}) C: {self.C:02X}({self.C})\n" +
            f"D: {self.D:02X}({self.D})  E: {self.E:02X}({self.E}) \n" +
            f"HL: {self.HL:04X}({self.HL}) SP: {self.SP:04X}({self.SP}) PC: {self.PC:04X}({self.PC})"
        )
        
    def dump_json(self):
        self.A &= 0xFF
        self.B &= 0xFF
        self.C &= 0xFF
        self.D &= 0xFF
        self.E &= 0xFF

        self.HL &= 0xFFFF
        self.SP &= 0xFFFF
        self.PC &= 0xFFFF
        return {
            "Name": self.func_name,
            "AF": f"{self.A:08b}|{self.F:08b} ({self.A:<3}|{self.F:>3})",
            "BC": f"{self.A:08b}|{self.F:08b} ({self.B:<3}|{self.C:>3})",
            "DE": f"{self.A:08b}|{self.F:08b} ({self.D:<3}|{self.E:>3})",
            "HL": f"{self.HL:016b}| ({self.HL:<5})",
            "SP": f"{self.HL:016b}| ({self.SP:<5})",
            "PC": f"{self.HL:016b}| ({self.PC:<5})",
            
        }
    
    def dump(self):
        with open("registers-validate.json" , "w") as f:
            self.A &= 0xFF
            self.B &= 0xFF
            self.C &= 0xFF
            self.D &= 0xFF
            self.E &= 0xFF
            self.HL &= 0xFFFF
            self.SP &= 0xFFFF
            self.PC &= 0xFFFF
            
            json.dump({
                "name": self.Name,
                "a": self.A,
                "f": self.F,
                "b": self.B,
                "c": self.C,
                "d": self.D,
                "e": self.E,
                "hl": self.HL,
                "sp": self.SP,
                "pc": self.PC,
                "args": str(self.args),
                "cycles": self.cycles
            }, f, indent=4)
        
rows = []
cpu = DummyCPU()
func_name = cpu.Name 

for f in dir(opcodes):
    try:
        n = int(f.split('_')[-1], 16)
        if n == cpu.Name:
            func_name = f
            break
    except:
        continue
    

# func_name = list(filter(lambda x: f"{cpu.Name:04x}" in x.split('_')[-1], dir(opcodes)))
func = getattr(opcodes, func_name)
if callable(func):
    try:
        cpu.cycles = func(cpu)
    except:
        cpu.cycles = func(cpu, cpu.args)
    print(cpu.report())
    rows.append(cpu.dump_json())
    cpu.dump()       
        
