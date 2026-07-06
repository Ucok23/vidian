<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let apiKeyInput = $state('');
  let hasKey = $state(false);
  let isSaving = $state(false);
  let saveMessage = $state('');

  // Provider config
  let provider = $state('anthropic'); // 'anthropic' | 'openai'
  let baseUrl = $state('');
  let model = $state('');
  let aiKeyInput = $state('');
  let hasAiKey = $state(false);

  async function loadStatus() {
    try {
      const res = await fetch('/api/settings');
      const data = await res.json();
      hasKey = !!data.hasKey;
      provider = data.aiProvider || 'anthropic';
      baseUrl = data.aiBaseUrl || '';
      model = data.aiModel || '';
      hasAiKey = !!data.hasAiKey;
    } catch (e) {
      // Settings status is best-effort; leave values as-is.
    }
  }

  $effect(() => {
    if (store.settingsOpen) {
      apiKeyInput = '';
      aiKeyInput = '';
      saveMessage = '';
      loadStatus();
    }
  });

  async function save() {
    isSaving = true;
    saveMessage = '';
    try {
      // Only send secret fields when the user typed something, so leaving them
      // blank keeps the existing value instead of clearing it.
      const body = {
        aiProvider: provider,
        aiBaseUrl: baseUrl,
        aiModel: model
      };
      if (apiKeyInput) body.anthropicApiKey = apiKeyInput;
      if (aiKeyInput) body.aiApiKey = aiKeyInput;

      const res = await fetch('/api/settings/save', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
      });
      const data = await res.json();
      hasKey = !!data.hasKey;
      hasAiKey = !!data.hasAiKey;
      apiKeyInput = '';
      aiKeyInput = '';
      saveMessage = 'Saved.';
    } catch (e) {
      saveMessage = 'Failed to save: ' + e.message;
    } finally {
      isSaving = false;
    }
  }

  function close() {
    store.settingsOpen = false;
  }
</script>

{#if store.settingsOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="overlay" onclick={close}>
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="panel" onclick={(e) => e.stopPropagation()}>
      <div class="panel-header">
        <Icon name="settings" size={18} />
        <h2>Settings</h2>
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <span class="close-btn" onclick={close}><Icon name="close" size={16} /></span>
      </div>

      <div class="panel-body">
        <div class="field-label">AI Provider</div>
        <p class="field-hint">
          Powers the onboarding narrator and “Explain” feature. Everything is stored
          locally on this machine and sent only to the endpoint you choose.
        </p>
        <div class="provider-toggle">
          <button class:active={provider === 'anthropic'} onclick={() => provider = 'anthropic'}>Anthropic</button>
          <button class:active={provider === 'openai'} onclick={() => provider = 'openai'}>OpenAI-compatible</button>
        </div>

        {#if provider === 'anthropic'}
          <div class="field-label">Anthropic API Key</div>
          <p class="field-hint">
            Sent only to Anthropic's API.
            {#if hasKey}<span class="status-ok">A key is currently configured.</span>{/if}
          </p>
          <input
            type="password"
            placeholder={hasKey ? 'Enter a new key to replace the current one' : 'sk-ant-...'}
            bind:value={apiKeyInput}
          />
          <div class="field-label mt">Model (optional)</div>
          <input type="text" placeholder="claude-sonnet-5 (default)" bind:value={model} />
        {:else}
          <p class="field-hint">
            Any OpenAI-compatible <code>/chat/completions</code> endpoint — OpenAI, or a
            local server like Ollama or LM Studio. Point the base URL at the API root
            (Vidian appends <code>/chat/completions</code>).
          </p>
          <div class="field-label">Base URL</div>
          <input type="text" placeholder="http://localhost:11434/v1" bind:value={baseUrl} />
          <div class="field-label mt">Model</div>
          <input type="text" placeholder="llama3.1, gpt-4o-mini, …" bind:value={model} />
          <div class="field-label mt">API Key (optional for local models)</div>
          <p class="field-hint">
            {#if hasAiKey}<span class="status-ok">A key is currently configured.</span>{/if}
          </p>
          <input
            type="password"
            placeholder={hasAiKey ? 'Enter a new key to replace the current one' : 'sk-… (leave blank for local)'}
            bind:value={aiKeyInput}
          />
        {/if}

        <div class="actions">
          <button class="save-btn" onclick={save} disabled={isSaving}>
            {isSaving ? 'Saving…' : 'Save'}
          </button>
          {#if saveMessage}<span class="save-message">{saveMessage}</span>{/if}
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .panel {
    width: 440px;
    max-width: 90vw;
    background: #1b1b20;
    border: 1px solid #2d2d34;
    border-radius: 10px;
    color: #e3e3e6;
    font-family: system-ui, sans-serif;
    box-shadow: 0 12px 40px rgba(0, 0, 0, 0.4);
  }

  .panel-header {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 16px 20px;
    border-bottom: 1px solid #2d2d34;
  }

  .panel-header h2 {
    font-size: 15px;
    font-weight: 700;
    margin: 0;
    flex: 1;
  }

  .close-btn {
    cursor: pointer;
    color: #8e8e93;
    display: flex;
  }

  .close-btn:hover {
    color: #e3e3e6;
  }

  .panel-body {
    padding: 20px;
  }

  .field-label {
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.4px;
    text-transform: uppercase;
    color: #8e8e93;
    margin-bottom: 8px;
  }

  .field-label.mt {
    margin-top: 16px;
  }

  .field-hint code {
    font-family: 'Fira Code', monospace;
    font-size: 11px;
    color: #a5b4fc;
    background: rgba(99, 102, 241, 0.1);
    padding: 1px 4px;
    border-radius: 3px;
  }

  .provider-toggle {
    display: flex;
    gap: 6px;
    margin-bottom: 18px;
  }

  .provider-toggle button {
    flex: 1;
    background: #121214;
    border: 1px solid #2d2d34;
    color: #8e8e93;
    border-radius: 6px;
    padding: 8px 10px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: border-color 0.12s, color 0.12s, background-color 0.12s;
  }

  .provider-toggle button:hover {
    color: #e3e3e6;
    border-color: #3d3d50;
  }

  .provider-toggle button.active {
    background: rgba(99, 102, 241, 0.15);
    border-color: #6366f1;
    color: #a5b4fc;
  }

  .field-hint {
    font-size: 12px;
    color: #8e8e93;
    line-height: 1.5;
    margin: 0 0 12px;
  }

  .status-ok {
    display: block;
    color: #4ade80;
    margin-top: 4px;
  }

  input {
    width: 100%;
    box-sizing: border-box;
    background: #121214;
    border: 1px solid #2d2d34;
    border-radius: 6px;
    padding: 8px 10px;
    color: #e3e3e6;
    font-size: 13px;
    font-family: 'Fira Code', monospace;
  }

  input:focus {
    outline: none;
    border-color: #6366f1;
  }

  .actions {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 14px;
  }

  .save-btn {
    background: #6366f1;
    color: white;
    border: none;
    border-radius: 6px;
    padding: 7px 16px;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
  }

  .save-btn:hover {
    background: #4f46e5;
  }

  .save-btn:disabled {
    opacity: 0.6;
    cursor: default;
  }

  .save-message {
    font-size: 12px;
    color: #8e8e93;
  }
</style>
