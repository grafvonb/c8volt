(() => {
  const commandTokens = new Set([
    "cancel",
    "capabilities",
    "cluster",
    "config",
    "delete",
    "deploy",
    "embed",
    "expect",
    "export",
    "get",
    "license",
    "list",
    "pd",
    "pi",
    "process-definition",
    "process-instance",
    "resource",
    "run",
    "show",
    "template",
    "tenant",
    "test-connection",
    "topology",
    "validate",
    "version",
    "walk",
  ]);

  const commandPattern = /\b[a-z][a-z-]*\b/g;
  const boundaryPattern = /[./\w-]/;
  const skippedClassPattern = /\b(c|ch|cm|cp|cpf|c1|cs|s|s1|s2|sb|sc|sd|sh|sx|nt|na|nb|nv)\b/;

  function shouldSkip(node) {
    for (let el = node.parentElement; el; el = el.parentElement) {
      if (el.tagName === "CODE") {
        return false;
      }

      if (skippedClassPattern.test(el.className || "")) {
        return true;
      }
    }

    return false;
  }

  function decorateTextNode(node) {
    const text = node.nodeValue;
    let cursor = 0;
    let changed = false;
    const fragment = document.createDocumentFragment();

    for (const match of text.matchAll(commandPattern)) {
      const token = match[0];
      const start = match.index;
      const end = start + token.length;
      const before = start > 0 ? text[start - 1] : "";
      const after = end < text.length ? text[end] : "";

      if (!commandTokens.has(token) || boundaryPattern.test(before) || boundaryPattern.test(after)) {
        continue;
      }

      fragment.append(document.createTextNode(text.slice(cursor, start)));

      const span = document.createElement("span");
      span.className = "c8volt-command-token";
      span.textContent = token;
      fragment.append(span);

      cursor = end;
      changed = true;
    }

    if (!changed) {
      return;
    }

    fragment.append(document.createTextNode(text.slice(cursor)));
    node.replaceWith(fragment);
  }

  function decorateCodeBlock(code) {
    if (!code.textContent.includes("c8volt")) {
      return;
    }

    const walker = document.createTreeWalker(code, NodeFilter.SHOW_TEXT, {
      acceptNode(node) {
        if (!node.nodeValue.trim() || shouldSkip(node)) {
          return NodeFilter.FILTER_REJECT;
        }

        return NodeFilter.FILTER_ACCEPT;
      },
    });

    const nodes = [];
    for (let node = walker.nextNode(); node; node = walker.nextNode()) {
      nodes.push(node);
    }

    nodes.forEach(decorateTextNode);
  }

  function decorateCodeBlocks() {
    document.querySelectorAll(".language-bash.highlighter-rouge code").forEach(decorateCodeBlock);
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", decorateCodeBlocks);
  } else {
    decorateCodeBlocks();
  }
})();
