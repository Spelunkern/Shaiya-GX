// Prints the C strings at the given addresses.
import ghidra.app.script.GhidraScript;
public class Str extends GhidraScript {
  @Override public void run() throws Exception {
    for (String a : getScriptArgs()) {
      var addr = toAddr(a);
      StringBuilder sb = new StringBuilder();
      for (int i = 0; i < 256; i++) { byte b = getByte(addr.add(i)); if (b == 0) break; sb.append((char) (b & 0xff)); }
      println("S " + a + ": \"" + sb + "\"");
    }
  }
}
