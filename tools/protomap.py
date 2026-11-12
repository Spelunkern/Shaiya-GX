#!/usr/bin/env python3
"""Regenerates docs/protocolo.md: every opcode the EP5.4 client handles
(dispatcher fun_0059cf70 in shaiya-client-decomp), its handler, the name
used here (internal/proto/opcodes.go), whether this client decodes it
(internal/game/state.go) and the verified client writers (internal/proto)."""
import os, re, sys, glob
root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
decomp = os.environ.get("SHAIYA_DECOMP", os.path.expanduser("~/src/shaiya-client-decomp"))
disp = open(os.path.join(decomp, "src/client/network/dispatcher.cpp")).read()
body = disp[disp.index("fun_0059cf70(uint32_t opcode"):]
cases = []
for m in re.finditer(r"^\s*case (0x[0-9a-f]+):([\s\S]*?)(?=^\s*(?:case|default)\b|\Z)", body, re.M):
    call = re.search(r"(\w+)\(", m.group(2))
    cases.append((int(m.group(1), 16), call.group(1) if call else "?"))
ops = open(os.path.join(root, "internal/proto/opcodes.go")).read()
names = {int(v, 16): n for n, v in re.findall(r"(\w+)\s+Opcode = (0x[0-9A-Fa-f]+)", ops)}
state = open(os.path.join(root, "internal/game/state.go")).read()
decoded = set()
for grp in re.findall(r"case (proto\.[\w, .]+):", state):
    for n in re.findall(r"proto\.(\w+)", grp):
        decoded |= {v for v, k in names.items() if k == n}
writers = "\n".join(open(p).read() for p in glob.glob(os.path.join(root,"internal/proto/*.go")) if not p.endswith("_test.go"))
sent = sorted(a for a in set(re.findall(r"\b(0x59[0-9a-f]{4})\b", writers)) if 0x597f60 < int(a, 16) < 0x59c300)
out = ["# Protocolo EP5.4", "",
       "Generado por `tools/protomap.py`; no editar a mano.", "",
       "- **Recibidos:** los %d opcodes que despacha el cliente original (`fun_0059cf70`, `network/dispatcher.cpp` del decomp)." % len(cases),
       "- **Decodificado:** sí = `internal/game` lo aplica; no = se cuenta y se ignora.",
       "- **Enviados:** los writers del original verificados en Ghidra (rango `0x597f60..0x59c300`): %s." % ", ".join(sent), "",
       "Recibidos: %d. Decodificados: %d." % (len(cases), sum(1 for o, _ in cases if o in decoded)), "",
       "| Opcode | Handler del cliente | Nombre | Decodificado |", "|---|---|---|---|"]
for o, h in cases:
    out.append("| 0x%04X | `%s` | %s | %s |" % (o, h, names.get(o, ""), "sí" if o in decoded else "no"))
open(os.path.join(root, "docs/protocolo.md"), "w").write("\n".join(out) + "\n")
print("cases", len(cases), "decoded", sum(1 for o, _ in cases if o in decoded))
