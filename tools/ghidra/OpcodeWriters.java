// Lists "MOV word ptr [...],IMM" stores whose immediate is one of the given
// opcodes (hex), i.e. candidate packet writers, with their function.
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.*;
import ghidra.program.model.scalar.Scalar;
import java.util.*;
public class OpcodeWriters extends GhidraScript {
  @Override public void run() throws Exception {
    Set<Long> want = new HashSet<>();
    for (String a : getScriptArgs()) want.add(Long.parseLong(a.replace("0x", ""), 16));
    for (Instruction ins : currentProgram.getListing().getInstructions(true)) {
      if (!ins.getMnemonicString().equals("MOV") || ins.getNumOperands() != 2) continue;
      String t = ins.toString();
      if (!t.startsWith("MOV word ptr")) continue;
      Object[] o = ins.getOpObjects(1);
      if (o.length != 1 || !(o[0] instanceof Scalar)) continue;
      long v = ((Scalar) o[0]).getUnsignedValue();
      if (!want.contains(v)) continue;
      Function f = getFunctionContaining(ins.getAddress());
      println(String.format("0x%04x %s %s %s", v, ins.getAddress(), f == null ? "?" : f.getName(), t));
    }
  }
}
