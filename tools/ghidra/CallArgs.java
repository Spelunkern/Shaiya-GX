// For every CALL to TARGET inside FUNC, reconstructs the first N stack
// arguments from the preceding PUSHes: CallArgs.java FUNC TARGET N
// Immediates print as hex/float, strings are resolved, and a register
// produced by ftol (CALL 0x5cd414) prints the x87 expression that fed it.
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.*;
import ghidra.program.model.scalar.Scalar;
import ghidra.program.model.address.Address;
import java.util.*;
public class CallArgs extends GhidraScript {
  Listing L;
  String flt(Address a) {
    try { return String.valueOf(Float.intBitsToFloat(getInt(a))); } catch (Exception e) { return "?"; }
  }
  String str(long v) {
    try {
      Address a = toAddr(v);
      StringBuilder sb = new StringBuilder();
      for (int i = 0; i < 80; i++) { byte b = getByte(a.add(i)); if (b == 0) break; if (b < 32 || b > 126) return null; sb.append((char) b); }
      return sb.length() >= 3 ? "\"" + sb + "\"" : null;
    } catch (Exception e) { return null; }
  }
  String imm(long v) {
    if (v >= 0x400000 && v < 0x800000) { String s = str(v); if (s != null) return s; }
    if (v > 0x30000000L && v < 0x50000000L) return Float.intBitsToFloat((int) v) + "f";
    return "0x" + Long.toHexString(v);
  }
  String operand(Instruction ins, int op) {
    Object[] o = ins.getOpObjects(op);
    String t = ins.getDefaultOperandRepresentation(op);
    if (o.length == 1 && o[0] instanceof Address && t.contains("[")) {
      return "[" + o[0] + "]=" + flt((Address) o[0]);
    }
    return t;
  }
  // x87 expression evaluated before a CALL ftol, walking back to its FLD/FILD.
  String fpExpr(Instruction call) {
    List<String> parts = new ArrayList<>();
    Instruction i = call.getPrevious();
    for (int k = 0; k < 12 && i != null; k++, i = i.getPrevious()) {
      String m = i.getMnemonicString();
      if (!m.startsWith("F")) continue;
      parts.add(0, m + " " + operand(i, 0));
      if (m.equals("FLD") || m.equals("FILD")) break;
    }
    return "ftol(" + String.join("; ", parts) + ")";
  }
  String regValue(Instruction push, String reg) {
    Instruction i = push.getPrevious();
    for (int k = 0; k < 60 && i != null; k++, i = i.getPrevious()) {
      String m = i.getMnemonicString();
      if (m.equals("CALL") && reg.equals("EAX")) {
        if (i.toString().contains("0x005cd414")) return fpExpr(i);
        return "ret(" + i.getDefaultOperandRepresentation(0) + ")";
      }
      if (i.getNumOperands() == 2 && i.getDefaultOperandRepresentation(0).equals(reg)) {
        if (m.equals("MOV")) {
          Object[] o = i.getOpObjects(1);
          if (o.length == 1 && o[0] instanceof Scalar) return imm(((Scalar) o[0]).getUnsignedValue());
          return reg + "<-" + i.getDefaultOperandRepresentation(1);
        }
        if (m.equals("LEA")) return reg + "<-lea " + i.getDefaultOperandRepresentation(1);
        if (m.equals("XOR") && i.getDefaultOperandRepresentation(1).equals(reg)) return "0x0";
        return reg + "<-" + i;
      }
    }
    return reg + "?";
  }
  @Override public void run() throws Exception {
    String[] a = getScriptArgs();
    Function f = getFunctionAt(toAddr(a[0]));
    long target = Long.parseLong(a[1].replace("0x", ""), 16);
    int n = Integer.parseInt(a[2]);
    L = currentProgram.getListing();
    for (Instruction ins : L.getInstructions(f.getBody(), true)) {
      if (!ins.getMnemonicString().equals("CALL")) continue;
      Address[] flows = ins.getFlows();
      if (flows.length != 1 || flows[0].getOffset() != target) continue;
      List<String> args = new ArrayList<>();
      Instruction i = ins.getPrevious();
      String ecx = regValue(ins, "ECX");
      while (i != null && args.size() < n) {
        if (i.getMnemonicString().equals("PUSH")) {
          Object[] o = i.getOpObjects(0);
          String t = i.getDefaultOperandRepresentation(0);
          if (o.length == 1 && o[0] instanceof Scalar) args.add(imm(((Scalar) o[0]).getUnsignedValue()));
          else if (t.matches("E[A-Z][A-Z]")) args.add(regValue(i, t));
          else args.add(t);
        }
        i = i.getPrevious();
      }
      println("== call @" + ins.getAddress() + " this=" + ecx);
      for (int k = 0; k < args.size(); k++) println("   a" + (k + 1) + " = " + args.get(k));
    }
  }
}
