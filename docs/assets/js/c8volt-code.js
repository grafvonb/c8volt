(() => {
  const tokenPattern = /(\s+|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|<[^>\s]+>|[|&;()]+|[^\s]+)/g;
  const operatorPattern = /^[|&;()]+$/;
  const globalValueFlags = new Set(["--config", "--profile", "--tenant", "--timeout"]);

  function span(className, text) {
    const el = document.createElement("span");
    el.className = className;
    el.textContent = text;
    return el;
  }

  function appendToken(fragment, className, text) {
    if (!className) {
      fragment.append(document.createTextNode(text));
      return;
    }

    fragment.append(span(className, text));
  }

  function isC8voltBinary(token) {
    return token === "c8volt" || token === "./c8volt";
  }

  function isWhitespace(token) {
    return /^\s+$/.test(token);
  }

  function isFlag(token) {
    return /^--[a-z0-9][a-z0-9-]*(=.*)?$/.test(token);
  }

  function flagName(token) {
    return token.split("=", 1)[0];
  }

  function isPlaceholder(token) {
    return /^<[^>\s]+>$/.test(token);
  }

  function decorateLine(line, fragment) {
    if (!line.trim()) {
      fragment.append(document.createTextNode(line));
      return;
    }

    if (line.trimStart().startsWith("#")) {
      fragment.append(span("c8v-muted", line));
      return;
    }

    let commandMode = false;
    let commandSeen = false;
    let expectingGlobalFlagValue = false;

    for (const match of line.matchAll(tokenPattern)) {
      const token = match[0];

      if (isWhitespace(token)) {
        fragment.append(document.createTextNode(token));
        continue;
      }

      if (isC8voltBinary(token)) {
        appendToken(fragment, "c8v-bin", token);
        commandMode = true;
        commandSeen = false;
        continue;
      }

      if (operatorPattern.test(token)) {
        appendToken(fragment, "c8v-muted", token);
        commandMode = false;
        commandSeen = false;
        continue;
      }

      if (isFlag(token)) {
        appendToken(fragment, "c8v-flag", token);
        if (commandSeen) {
          commandMode = false;
        } else {
          expectingGlobalFlagValue = !token.includes("=") && globalValueFlags.has(flagName(token));
        }
        continue;
      }

      if (isPlaceholder(token)) {
        appendToken(fragment, "c8v-placeholder", token);
        if (expectingGlobalFlagValue && !commandSeen) {
          expectingGlobalFlagValue = false;
        } else {
          commandMode = false;
        }
        continue;
      }

      if (expectingGlobalFlagValue && !commandSeen) {
        fragment.append(document.createTextNode(token));
        expectingGlobalFlagValue = false;
        continue;
      }

      if (commandMode) {
        appendToken(fragment, "c8v-command", token);
        commandSeen = true;
        continue;
      }

      fragment.append(document.createTextNode(token));
    }
  }

  function decorateCodeBlock(code) {
    if (code.classList.contains("c8v-highlighted")) {
      return;
    }

    if (!code.textContent.includes("c8volt")) {
      return;
    }

    const fragment = document.createDocumentFragment();
    const lines = code.textContent.split("\n");

    lines.forEach((line, index) => {
      if (index > 0) {
        fragment.append(document.createTextNode("\n"));
      }

      decorateLine(line, fragment);
    });

    code.replaceChildren(fragment);
    code.classList.add("c8v-highlighted");
  }

  function decorateCodeBlocks() {
    document.querySelectorAll(".highlighter-rouge code").forEach(decorateCodeBlock);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", decorateCodeBlocks);
  } else {
    decorateCodeBlocks();
  }
})();
