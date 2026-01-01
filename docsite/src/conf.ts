const libName = "goinfer";
const libTitle = "Goinfer";
const repoUrl = "https://github.com/LynxAIeu/goinfer";

const links: Array<{ href: string; name: string }> = [
  // { href: "/python", name: "Python api" },
];

// python runtime
const examplesExtension = "js";

async function loadHljsTheme(isDark: boolean) {
  if (isDark) {
    await import("../node_modules/highlight.js/styles/base16/material-darker.css")
  } else {
    await import("../node_modules/highlight.js/styles/stackoverflow-light.css")
  }
}

/** Import the languages you need for highlighting */
import hljs from 'highlight.js/lib/core';
import go from 'highlight.js/lib/languages/go';
import yaml from 'highlight.js/lib/languages/yaml';
import bash from 'highlight.js/lib/languages/bash';
import typescript from 'highlight.js/lib/languages/typescript';
//import xml from 'highlight.js/lib/languages/xml';
//import json from 'highlight.js/lib/languages/json';
import javascript from "highlight.js/lib/languages/javascript";
hljs.registerLanguage('go', go);
hljs.registerLanguage('yaml', yaml);
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('typescript', typescript);
hljs.registerLanguage('bash', bash);
//hljs.registerLanguage('html', xml);
//hljs.registerLanguage('json', json);

export {
  libName,
  libTitle,
  repoUrl,
  examplesExtension,
  links,
  hljs,
  loadHljsTheme
}