import hljs from 'highlight.js/lib/core';
import typescript from 'highlight.js/lib/languages/typescript';
import JSON from 'highlight.js/lib/languages/json';
import javascript from 'highlight.js/lib/languages/javascript';

hljs.registerLanguage('typescript', typescript);
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('json', JSON);

export { hljs }