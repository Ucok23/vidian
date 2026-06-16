<script>
  import { onMount, tick } from 'svelte';
  import { marked } from 'marked';
  import mermaid from 'mermaid';

  let { content = '' } = $props();

  let previewContainer = $state(null);
  let renderedHtml = $state('');

  // Initialize Mermaid for dark theme
  mermaid.initialize({
    startOnLoad: false,
    theme: 'dark',
    securityLevel: 'loose',
    logLevel: 5
  });

  // Custom renderer for marked to wrap code block with mermaid class
  const renderer = new marked.Renderer();
  
  // Save original code renderer
  const originalCode = renderer.code.bind(renderer);
  
  renderer.code = function({ text, lang, escaped }) {
    if (lang === 'mermaid') {
      return `<pre class="mermaid">${text}</pre>`;
    }
    return originalCode({ text, lang, escaped });
  };

  marked.setOptions({ renderer });

  // Watch content changes
  $effect(() => {
    try {
      renderedHtml = marked.parse(content || '');
    } catch (e) {
      console.error("Markdown parsing failed", e);
      renderedHtml = `<p style="color: #ef4444;">Failed to compile markdown: ${e.message}</p>`;
    }

    // Run Mermaid after DOM update
    tick().then(() => {
      if (previewContainer) {
        try {
          const mermaidNodes = previewContainer.querySelectorAll('.mermaid');
          if (mermaidNodes.length > 0) {
            // Re-render mermaid elements
            mermaid.run({
              nodes: mermaidNodes
            });
          }
        } catch (err) {
          console.error("Mermaid rendering failed", err);
        }
      }
    });
  });
</script>

<div bind:this={previewContainer} class="markdown-body">
  {@html renderedHtml}
</div>

<style>
  .markdown-body {
    padding: 24px;
    color: #e3e3e6;
    background-color: #1e1e24;
    font-size: 15px;
    line-height: 1.6;
    overflow-y: auto;
    height: 100%;
    width: 100%;
    box-sizing: border-box;
  }

  /* Markdown Elements styling (similar to github dark theme) */
  :global(.markdown-body h1, .markdown-body h2, .markdown-body h3, .markdown-body h4) {
    margin-top: 24px;
    margin-bottom: 16px;
    font-weight: 600;
    line-height: 1.25;
    color: #ffffff;
    border-bottom: 1px solid #2d2d34;
    padding-bottom: 0.3em;
  }

  :global(.markdown-body h1) { font-size: 2em; }
  :global(.markdown-body h2) { font-size: 1.5em; }
  :global(.markdown-body h3) { font-size: 1.25em; }
  
  :global(.markdown-body p) {
    margin-top: 0;
    margin-bottom: 16px;
    color: #c9d1d9;
  }

  :global(.markdown-body a) {
    color: #58a6ff;
    text-decoration: none;
  }

  :global(.markdown-body a:hover) {
    text-decoration: underline;
  }

  :global(.markdown-body code) {
    padding: 0.2em 0.4em;
    margin: 0;
    font-size: 85%;
    background-color: rgba(110, 118, 129, 0.4);
    border-radius: 6px;
    font-family: monospace;
  }

  :global(.markdown-body pre) {
    padding: 16px;
    overflow: auto;
    font-size: 85%;
    line-height: 1.45;
    background-color: #16161a;
    border-radius: 6px;
    margin-bottom: 16px;
  }

  :global(.markdown-body pre code) {
    background-color: transparent;
    padding: 0;
    margin: 0;
    font-size: 100%;
    word-break: normal;
    white-space: pre;
    direction: ltr;
    border: 0;
  }

  :global(.markdown-body blockquote) {
    padding: 0 1em;
    color: #8b949e;
    border-left: 0.25em solid #30363d;
    margin: 0 0 16px 0;
  }

  :global(.markdown-body ul, .markdown-body ol) {
    margin-top: 0;
    margin-bottom: 16px;
    padding-left: 2em;
  }

  :global(.markdown-body li) {
    margin-top: 0.25em;
  }

  :global(.markdown-body table) {
    display: block;
    width: 100%;
    width: max-content;
    max-width: 100%;
    overflow: auto;
    margin-top: 0;
    margin-bottom: 16px;
    border-spacing: 0;
    border-collapse: collapse;
  }

  :global(.markdown-body table th, .markdown-body table td) {
    padding: 6px 13px;
    border: 1px solid #30363d;
  }

  :global(.markdown-body table tr) {
    background-color: #1e1e24;
    border-top: 1px solid #21262d;
  }

  :global(.markdown-body table tr:nth-child(2n)) {
    background-color: #16161a;
  }

  /* Custom Mermaid styling to fit the dark theme */
  :global(.markdown-body .mermaid) {
    background: #16161a;
    display: flex;
    justify-content: center;
    padding: 16px;
    border-radius: 8px;
    border: 1px solid #2d2d34;
    margin-bottom: 16px;
    overflow-x: auto;
  }
</style>
