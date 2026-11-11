// Prints N little-endian floats at an address: Floats.java <address> <count>.
import ghidra.app.script.GhidraScript;
public class Floats extends GhidraScript {
  @Override public void run() throws Exception {
    String[] a = getScriptArgs();
    var addr = toAddr(a[0]);
    int n = Integer.parseInt(a[1]);
    StringBuilder sb = new StringBuilder("F " + a[0] + ":");
    for (int i = 0; i < n; i++) sb.append(" ").append(Float.intBitsToFloat(getInt(addr.add(4L * i))));
    println(sb.toString());
  }
}
