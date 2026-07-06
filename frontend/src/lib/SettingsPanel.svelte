<script>
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  let isSaving = $state(false);
  let saveMessage = $state('');

  // The four known providers. `key`/`base`/`model` flag which fields each one
  // uses; `keyOptional` marks a key that can be blank (local models).
  const PROVIDERS = [
    { id: 'anthropic',         label: 'Anthropic',          key: true,  base: false, model: true,  keyPlaceholder: 'sk-ant-…',    modelPlaceholder: 'claude-sonnet-5 (default)' },
    { id: 'openai',            label: 'OpenAI',             key: true,  base: false, model: true,  keyPlaceholder: 'sk-…',        modelPlaceholder: 'gpt-4o-mini' },
    { id: 'openai-compatible', label: 'OpenAI-compatible',  key: true,  base: true,  model: true,  keyOptional: true, keyPlaceholder: 'sk-… (blank for local)', modelPlaceholder: 'llama3.1, …', basePlaceholder: 'http://localhost:11434/v1' },
    { id: 'gemini',            label: 'Google Gemini',      key: true,  base: false, model: true,  keyPlaceholder: 'AIza…',       modelPlaceholder: 'gemini-2.0-flash (default)' },
  ];

  let active = $state('anthropic');
  // Per-provider field state, keyed by provider id.
  let baseUrl = $state({});
  let model = $state({});
  let keyInput = $state({});   // typed-in secret (never populated from server)
  let hasKey = $state({});     // whether a key is already saved server-side

  const current = $derived(PROVIDERS.find(p => p.id === active) || PROVIDERS[0]);

  async function loadStatus() {
    try {
      const res = await fetch('/api/settings');
      const data = await res.json();
      active = data.activeProvider || 'anthropic';
      const provs = data.providers || {};
      const bu = {}, md = {}, hk = {}, ki = {};
      for (const p of PROVIDERS) {
        const c = provs[p.id] || {};
        bu[p.id] = c.baseUrl || '';
        md[p.id] = c.model || '';
        hk[p.id] = !!c.hasKey;
        ki[p.id] = '';
      }
      baseUrl = bu; model = md; hasKey = hk; keyInput = ki;
    } catch (e) {
      // Settings status is best-effort; leave values as-is.
    }
  }

  $effect(() => {
    if (store.settingsOpen) {
      saveMessage = '';
      loadStatus();
    }
  });

  async function save() {
    isSaving = true;
    saveMessage = '';
    try {
      // Only send the API key when the user typed one, so leaving it blank
      // keeps the stored value instead of clearing it.
      const provider = {
        baseUrl: baseUrl[active] || '',
        model: model[active] || ''
      };
      if (keyInput[active]) provider.apiKey = keyInput[active];

      const res = await fetch('/api/settings/save', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ activeProvider: active, provider })
      });
      if (!res.ok) {
        saveMessage = 'Failed: ' + (await res.text()).trim();
        return;
      }
      const data = await res.json();
      const provs = data.providers || {};
      for (const p of PROVIDERS) {
        hasKey[p.id] = !!(provs[p.id] && provs[p.id].hasKey);
      }
      keyInput[active] = '';
      saveMessage = data.configured ? 'Saved — provider ready.' : 'Saved (missing required fields).';
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
          Powers the onboarding narrator and “Explain” feature. Configure any of the
          providers below; the selected one is used. Everything is stored locally on
          this machine and sent only to the endpoint you choose.
        </p>
        <div class="provider-grid">
          {#each PROVIDERS as p (p.id)}
            <button
              class:active={active === p.id}
              onclick={() => active = p.id}
            >
              {p.label}
              {#if hasKey[p.id] || baseUrl[p.id]}<span class="dot" title="Configured"></span>{/if}
            </button>
          {/each}
        </div>

        {#if current.base}
          <div class="field-label">Base URL</div>
          <p class="field-hint">
            Any OpenAI-compatible <code>/chat/completions</code> endpoint (Ollama, LM
            Studio, vLLM…). Point at the API root; Vidian appends the path.
          </p>
          <input type="text" placeholder={current.basePlaceholder} bind:value={baseUrl[active]} />
        {/if}

        {#if current.model}
          <div class="field-label" class:mt={current.base}>Model</div>
          <input type="text" placeholder={current.modelPlaceholder} bind:value={model[active]} />
        {/if}

        {#if current.key}
          <div class="field-label mt">
            API Key{#if current.keyOptional} <span class="soft">(optional for local)</span>{/if}
          </div>
          <p class="field-hint">
            {#if hasKey[active]}<span class="status-ok">A key is currently configured.</span>{/if}
          </p>
          <input
            type="password"
            placeholder={hasKey[active] ? 'Enter a new key to replace the current one' : current.keyPlaceholder}
            bind:value={keyInput[active]}
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

  .provider-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 6px;
    margin-bottom: 18px;
  }

  .provider-grid button {
    position: relative;
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

  .provider-grid button:hover {
    color: #e3e3e6;
    border-color: #3d3d50;
  }

  .provider-grid button.active {
    background: rgba(99, 102, 241, 0.15);
    border-color: #6366f1;
    color: #a5b4fc;
  }

  .provider-grid .dot {
    position: absolute;
    top: 7px;
    right: 7px;
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #4ade80;
  }

  .soft {
    font-weight: 400;
    text-transform: none;
    letter-spacing: 0;
    color: #6b7280;
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
