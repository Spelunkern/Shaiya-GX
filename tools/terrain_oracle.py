#!/usr/bin/env python3
"""Verify EP5 terrain height and finest mesh using isolated original x86 routines.

Optional analysis dependencies: pefile and unicorn. No game initialization, OS,
network or graphics calls run. The only replaced operation is heap allocation
for terrain nodes; all selection, topology and height math execute unchanged.
Outputs are local evidence, not redistributable original game assets.
"""
import argparse
import hashlib
import json
import random
import struct
from pathlib import Path

import pefile
from unicorn import Uc, UC_ARCH_X86, UC_MODE_32, UC_HOOK_CODE
from unicorn.x86_const import UC_X86_REG_ESP, UC_X86_REG_ECX, UC_X86_REG_EIP, UC_X86_REG_EAX

SHA256 = '8e5ef14a2a3bfef15bae61b03d185e551b6c3fd44d09a2861d57428400bb3263'
STACK, STOP = 0x4080000, 0x3090000


class Native:
    def __init__(self, executable):
        raw = executable.read_bytes()
        if hashlib.sha256(raw).hexdigest() != SHA256:
            raise ValueError('Unsupported original executable hash')
        image = pefile.PE(data=raw).get_memory_mapped_image()
        self.u = Uc(UC_ARCH_X86, UC_MODE_32)
        self.u.mem_map(0x400000, (len(image) + 4095) & ~4095)
        self.u.mem_write(0x400000, image)
        self.u.mem_map(0x3000000, 0x100000)
        self.u.mem_map(0x4000000, 0x100000)

    def ints(self, address, *values):
        self.u.mem_write(address, struct.pack('<' + 'I' * len(values), *values))

    def run(self, start, end, esp, ecx):
        self.u.reg_write(UC_X86_REG_ESP, esp)
        self.u.reg_write(UC_X86_REG_ECX, ecx)
        hook = self.u.hook_add(UC_HOOK_CODE, lambda u, a, s, d: u.emu_stop(), begin=end, end=end)
        try:
            self.u.emu_start(start, 0, count=500000)
        finally:
            self.u.hook_del(hook)
        if self.u.reg_read(UC_X86_REG_EIP) != end:
            raise AssertionError('Original routine did not return')


parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('executable', type=Path)
parser.add_argument('--data', type=Path, required=True, help='Loose client data directory (read-only)')
parser.add_argument('--output-dir', type=Path, required=True)
args = parser.parse_args()
args.output_dir.mkdir(parents=True, exist_ok=True)
n=Native(args.executable)
obj,data,out=0x3000000,0x3050000,0x3080000
n.u.mem_write(STOP,b'\xd9\x1d'+struct.pack('<I',out))
n.ints(obj,data)
def bits(v):return struct.unpack('<I',struct.pack('<f',v))[0]
def native(vals,side,x,z):
 n.ints(0x224e650,side);n.u.mem_write(data,struct.pack('<'+'H'*len(vals),*vals))
 n.ints(STACK,STOP,bits(x/2),bits(z/2))
 n.run(0x5b3720,STOP+6,esp=STACK,ecx=obj)
 return struct.unpack('<f',n.u.mem_read(out,4))[0]
def fan(v,side,x,z):
 x/=2;z/=2;ix=min(int(x)//4*4,side-5);iz=min(int(z)//4*4,side-5);dx=x-ix;dz=z-iz
 a=v[iz*side+ix];b=v[iz*side+ix+4];c=v[(iz+4)*side+ix];d=v[(iz+4)*side+ix+4];m=v[(iz+2)*side+ix+2]
 if dz<=dx:
  if 4-dx<dz:return b+(4-dx)*(m-b)/2+(dz-(4-dx))*(d-b)/4
  return a+dx*(b-a)/4+dz*(m-(a+b)/2)/2
 if 4-dx<dz:return c+dx*(d-c)/4+(4-dz)*(m-(c+d)/2)/2
 return a+dz*(c-a)/4+dx*(m-(a+c)/2)/2
rng=random.Random(0x5b3720);fixtures=[];count=0;err=0
for i in range(8):
 side=17;v=[10000]* (side*side)
 if i==1:v[side+1]=65535
 if i==2:v[2*side+2]=30000
 if i==3:v[2]=65535
 if i>=4:v=[rng.randrange(0,60000) for _ in v]
 q=[]
 for j in range(125):
  x,z=rng.uniform(.01,31.99),rng.uniform(.01,31.99)
  h=native(v,side,x,z);want=fan(v,side,x,z);err=max(err,abs(h-want));count+=1
  assert abs(h-want)<.025,(x,z,h,want)
  q.append([x,z,(h-10000)/50])
 fixtures.append(dict(heights=v,queries=q))
report=dict(executable_sha256=hashlib.sha256(args.executable.read_bytes()).hexdigest(),routine='0x5b3720',synthetic_queries=count,max_raw_error=err,fixtures=fixtures)
(args.output_dir/'height-oracle.json').write_text(json.dumps(report))
print('PASS',count,'native queries; maximum raw error',err)
# Read real FLD headers only, preserving assets.
worlds=[]
real_error=0.0
for p in sorted((args.data/'world').glob('*.wld')):
 raw=p.read_bytes()
 if raw[:3]!=b'FLD':continue
 size=struct.unpack_from('<I',raw,4)[0];side=size//2+1
 vals=struct.unpack_from('<'+'H'*(side*side),raw,8)
 # A random set of points and source lattice vertices captures both spikes and valleys.
 diff=[];probes=[]
 for j in range(256):
  ix=rng.randrange(side-1);iz=rng.randrange(side-1);x=ix*2;z=iz*2
  expected=fan(vals,side,x,z);diff.append((abs(vals[iz*side+ix]-expected)/50,x,z,(vals[iz*side+ix]-10000)/50,(expected-10000)/50))
 # Compare native height on an isolated 32-unit crop with identical original samples.
 start=(side//2//4-2)*4
 crop=[vals[(start+z)*side+start+x] for z in range(17) for x in range(17)]
 for j in range(16):
  x,z=rng.uniform(.01,31.99),rng.uniform(.01,31.99);actual=native(crop,17,x,z);want=fan(crop,17,x,z)
  assert abs(actual-want)<.03
  real_error=max(real_error,abs(actual-want))
 worlds.append(dict(map=p.stem,size=size,largest_sample_differences=sorted(diff,reverse=True)[:4]))
(args.output_dir/'real-maps.json').write_text(json.dumps(worlds,indent=2))
print('PASS',len(worlds),'real field crops; maximum raw error',real_error)

from unicorn import UC_HOOK_CODE
from unicorn.x86_const import UC_X86_REG_ESP,UC_X86_REG_EAX,UC_X86_REG_EIP
n.u.mem_map(0x3100000,0x100000)
block,heights,assign,layers,buffers,heap,sentinel=0x3000000,0x3050000,0x3060000,0x3040000,0x3100000,0x3070000,0x3081000
n.u.mem_write(block,bytes(0x138));n.u.mem_write(sentinel,bytes(0x34))
def read(a):return struct.unpack('<I',n.u.mem_read(a,4))[0]
def run(a,args=(),ecx=block):
 n.ints(STACK,STOP,*args);n.run(a,STOP,esp=STACK,ecx=ecx)
def alloc(u,a,size,d):
 global heap
 sp=u.reg_read(UC_X86_REG_ESP);assert read(sp+4)==0x34
 u.mem_write(heap,bytes(0x34));u.reg_write(UC_X86_REG_EAX,heap);heap+=0x40
 u.reg_write(UC_X86_REG_EIP,read(sp));u.reg_write(UC_X86_REG_ESP,sp+4)
n.u.hook_add(UC_HOOK_CODE,alloc,begin=0x5cc96d,end=0x5cc96d)
for a,v in [(0x224e650,17),(0x224e6a4,16),(0x224e69c,7),(0x224e6a0,32),(0x224e6ac,sentinel),(0x224e6b0,sentinel),(0x224e500,layers),(0x224e508,1)]:n.ints(a,v)
for a,v in [(0x224e400,75),(0x224e404,1024),(0x224e408,80),(0x224e668,8),(0x224e670,8)]:n.ints(a,bits(v))
vals=[10000]*289;vals[18]=65535;vals[36]=14000
n.u.mem_write(heights,struct.pack('<289H',*vals))
run(0x5b4f90,[heights,assign,0,0,0]);run(0x5b5740)
n.u.mem_write(block+0xc4,b'\x01');n.u.mem_write(block+0xd0,b'\x01')
run(0x5b77b0);run(0x5b4de0)
n.ints(layers+0x31c,buffers,buffers+0x20000,0,0,0)
run(0x5b6160)
vc,ic=read(layers+0x324),read(layers+0x32c)
vertices=[struct.unpack('<3f',n.u.mem_read(buffers+i*36,12)) for i in range(vc)]
indices=struct.unpack('<'+'H'*ic,n.u.mem_read(buffers+0x20000,ic*2))
triangles=[sorted([vertices[indices[i+j]] for j in range(3)]) for i in range(0,ic,3)]
expected=[]
for z in range(0,16,4):
 for x in range(0,16,4):
  a=(x,vals[z*17+x],z);b=(x+4,vals[z*17+x+4],z);c=(x,vals[(z+4)*17+x],z+4);d=(x+4,vals[(z+4)*17+x+4],z+4);m=(x+2,vals[(z+2)*17+x+2],z+2)
  expected.extend(sorted(t) for t in [(a,b,m),(b,d,m),(d,c,m),(c,a,m)])
assert sorted(triangles)==sorted(expected),(vc,ic,len(expected))
(args.output_dir/'mesh-oracle.json').write_text(json.dumps(dict(heights=vals,vertices=vertices,indices=indices,triangles=ic//3,scope='Native near-camera selection and mesh emission; allocator hook supplies storage only. Synthetic unused spike and retained centre.')))
print('PASS native near mesh:',vc,'vertices,',ic//3,'triangles; exact five-point fan topology')
