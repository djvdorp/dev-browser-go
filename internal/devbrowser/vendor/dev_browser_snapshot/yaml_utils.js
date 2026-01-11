// === yaml ===
function yamlEscapeKeyIfNeeded(str) {
  if (!yamlStringNeedsQuotes(str)) return str;
  return "'" + str.replace(/'/g, "''") + "'";
}

function yamlEscapeValueIfNeeded(str) {
  if (!yamlStringNeedsQuotes(str)) return str;
  return '"' + str.replace(/[\\\"\x00-\x1f\x7f-\x9f]/g, c => {
    if (c === "\\") return "\\\\";
    if (c === '"') return '\\"';
    if (c === "\b") return "\\b";
    if (c === "\f") return "\\f";
    if (c === "\n") return "\\n";
    if (c === "\r") return "\\r";
    if (c === "\t") return "\\t";
    const code = c.charCodeAt(0);
    return "\\x" + code.toString(16).padStart(2, "0");
  }) + '"';
}

function yamlStringNeedsQuotes(str) {
  if (str.length === 0) return true;
  if (/^\s|\s$/.test(str)) return true;
  if (/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f-\x9f]/.test(str)) return true;
  if (/^-/.test(str)) return true;
  if (/[\n:](\s|$)/.test(str)) return true;
  if (/\s#/.test(str)) return true;
  if (/[\n\r]/.test(str)) return true;
  if (/^[&*\],?!>|@"'#%]/.test(str)) return true;
  if (/[{}\`]/.test(str)) return true;
  if (/^\[/.test(str)) return true;
  if (!Number.isNaN(Number(str)) || ["y","n","yes","no","true","false","on","off","null"].includes(str.toLowerCase())) return true;
  return false;
}
