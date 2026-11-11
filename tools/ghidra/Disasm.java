// Disassembles (in memory, read-only project) and prints N instructions
// from an address that may not belong to a function: Disasm.java ADDR N
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.Instruction;
public class Disasm extends GhidraScript {
  @Override public void run() throws Exception {
    var a = toAddr(getScriptArgs()[0]);
    int n = Integer.parseInt(getScriptArgs()[1]);
    disassemble(a);
    Instruction ins = getInstructionAt(a);
    for (int i = 0; i < n && ins != null; i++) {
      println("  " + ins.getAddress() + " " + ins);
      ins = ins.getNext();
    }
  }
}
