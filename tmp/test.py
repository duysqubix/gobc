from array import array 
from ctypes import c_void_p

def make_buffer(w, h):
    buf = array("B", [0x55] * (w*h*4))
    view = memoryview(buf).cast("I")
    buf0 = [view[i:i + w] for i in range(0, w * h, w)]
    buf_p = c_void_p(buf.buffer_info()[0])
    return buf0, buf_p

buf0, ptr = make_buffer(10, 5)
for v in buf0:
    print("v ", len(v))
    
