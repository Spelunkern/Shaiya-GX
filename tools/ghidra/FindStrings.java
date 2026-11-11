// Prints defined strings matching a regex (case-insensitive) and the
// functions that reference them: FindStrings.java REGEX
import ghidra.app.script.GhidraScript;
import ghidra.program.model.listing.*;
import ghidra.program.model.symbol.Reference;
import java.util.regex.Pattern;
public class FindStrings extends GhidraScript {
  @Override public void run() throws Exception {
    Pattern p = Pattern.compile(getScriptArgs()[0], Pattern.CASE_INSENSITIVE);
    DataIterator it = currentProgram.getListing().getDefinedData(true);
    while (it.hasNext()) {
      Data d = it.next();
      if (!d.hasStringValue()) continue;
      String s = String.valueOf(d.getValue());
      if (!p.matcher(s).find()) continue;
      StringBuilder refs = new StringBuilder();
      for (Reference r : getReferencesTo(d.getAddress())) {
        Function f = getFunctionContaining(r.getFromAddress());
        refs.append(" ").append(r.getFromAddress()).append("@").append(f == null ? "?" : f.getName());
      }
      println(d.getAddress() + " \"" + s + "\"" + refs);
    }
  }
}
