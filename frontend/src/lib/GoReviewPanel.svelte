<script>
  import { onMount } from 'svelte';
  import { store } from './store.svelte.js';
  import Icon from './Icon.svelte';

  const gr = $derived(store.goReview);
  const report = $derived(gr?.data);

  onMount(() => {
    if (!store.goReview) store.runGoReview();
  });
</script>

<div class="gorev">
  <div class="gorev-header">
    <div class="title-row">
      <Icon name="shieldCheck" size={20} color="#818cf8" />
      <h1>Go Review</h1>
      <span class="tagline">Deterministic findings — no AI, no judgment calls</span>
    </div>
    <button class="run-btn" onclick={() => store.runGoReview()} disabled={gr?.loading}>
      <Icon name="refresh" size={13} />
      {gr?.loading ? 'Analyzing…' : 'Re-run'}
    </button>
  </div>

  <div class="gorev-body">
    {#if gr?.loading && !report}
      <div class="state-msg">Analyzing module…</div>
    {:else if gr?.error}
      <div class="state-msg error">{gr.error}</div>
    {:else if report}
      <!-- Repo map (episode 02) -->
      <section class="card">
        <div class="card-head">
          <h2>Repo map</h2>
          <p class="what">
            What this checks: <code>go.mod</code> (module path, Go version, direct dependency count),
            <code>main</code> packages, the internal import graph ordered by dependency depth,
            the five largest non-test files, and the test/non-test line ratio. Pure <code>go list</code>
            + file-system reads — no parsing of business logic.
          </p>
        </div>

        <div class="kv-grid">
          <div class="kv"><span class="k">module</span><code class="v">{report.module.modulePath}</code></div>
          <div class="kv"><span class="k">go directive</span><code class="v">{report.module.goVersion || '—'}</code></div>
          <div class="kv"><span class="k">direct requires</span><code class="v">{report.module.directRequireCount}</code></div>
        </div>

        <div class="subhead">Entrypoints</div>
        {#if report.repoMap.entrypoints?.length}
          <div class="chip-row">
            {#each report.repoMap.entrypoints as e}
              <code class="chip">{e.dir || '.'}</code>
            {/each}
          </div>
        {:else}
          <div class="muted">No <code>main</code> package found.</div>
        {/if}

        <div class="subhead">Layers — {report.repoMap.layers?.length || 0} internal packages, ordered by fewest internal deps first</div>
        <table class="tbl">
          <thead><tr><th>package</th><th title="internal packages that import this one">fan-in</th><th title="internal packages this one imports">fan-out</th></tr></thead>
          <tbody>
            {#each (report.repoMap.layers || []) as p}
              <tr>
                <td><code class="pkg">{p.importPath}</code></td>
                <td class="num">{p.fanIn}</td>
                <td class="num">{p.fanOut}</td>
              </tr>
            {/each}
          </tbody>
        </table>

        <div class="two-col">
          <div>
            <div class="subhead">Hot files (largest non-test)</div>
            {#each (report.repoMap.hotFiles || []) as f}
              <div class="row"><code>{f.path}</code><span class="badge">{f.lines}</span></div>
            {/each}
          </div>
          <div>
            <div class="subhead">Test ratio</div>
            <div class="row muted">{report.repoMap.testLines} test lines / {report.repoMap.nonTestLines} non-test lines</div>
          </div>
        </div>

        {#if report.repoMap.unknowns?.length}
          <div class="subhead warn">Unknowns — the honest output of a first pass</div>
          {#each report.repoMap.unknowns as u}
            <div class="row unknown">{u}</div>
          {/each}
        {/if}
      </section>

      <!-- Package boundaries (episode 03) -->
      <section class="card">
        <div class="card-head">
          <h2>Package boundaries <span class="count">{report.packageFindings?.length || 0}</span></h2>
          <p class="what">
            What this checks: two mechanical smells over the import graph. <b>Landfill candidates</b> —
            packages named <code>utils</code>/<code>common</code>/<code>helpers</code>/<code>shared</code>/<code>lib</code>
            that are imported by 2+ internal packages and import nothing internal themselves (high fan-in, zero fan-out).
            <b>Inverted edges</b> — a package whose path looks like a domain layer (<code>domain</code>/<code>model</code>/<code>models</code>/<code>entity</code>)
            importing an HTTP/DB concern (<code>net/http</code>, <code>database/sql</code>, a SQL driver, a web framework).
            Every finding names the exact package and, for inverted edges, the exact import — flagged as a <i>candidate</i> to check, not a verdict.
          </p>
        </div>
        {#if !report.packageFindings?.length}
          <div class="muted">No landfill candidates or inverted edges found.</div>
        {:else}
          {#each report.packageFindings as f}
            <div class="finding">
              <span class="tag" class:landfill={f.category === 'landfill'} class:inverted={f.category === 'inverted-edge'}>{f.category}</span>
              <div class="finding-msg">{f.message}</div>
              <div class="finding-detail">{f.detail}</div>
            </div>
          {/each}
        {/if}
      </section>

      <!-- Interfaces at the wrong end (episode 04) -->
      <section class="card">
        <div class="card-head">
          <h2>Interfaces at the wrong end <span class="count">{(report.interfaces || []).filter(i => i.flagged).length}/{report.interfaces?.length || 0} flagged</span></h2>
          <p class="what">
            What this checks: every interface declared in the module, matched against every method set in the
            module by method name (an approximation — it doesn't type-check full signatures). A row is flagged
            when there's exactly <b>one implementation, declared in the same package as the interface itself</b> —
            the course's signature for "interface declared next to its only implementation," which mirrors the
            struct instead of abstracting it and forces every consumer to depend on the full method set.
          </p>
        </div>
        {#if !report.interfaces?.length}
          <div class="muted">No interfaces declared in this module.</div>
        {:else}
          {#each report.interfaces as i}
            <div class="finding" class:flagged={i.flagged}>
              <div class="finding-msg">
                <code>{i.name}</code> <span class="muted">in {i.package}</span>
                <span class="badge">{i.file}:{i.line}</span>
              </div>
              <div class="finding-detail">
                {i.methodCount} method(s) · {i.implementations?.length || 0} implementation(s){i.implementations?.length ? ': ' + i.implementations.join(', ') : ''}
              </div>
              {#if i.flagged}
                <div class="finding-detail flag-reason">{i.reason}</div>
              {/if}
            </div>
          {/each}
        {/if}
      </section>

      <!-- Context propagation (episode 07) -->
      <section class="card">
        <div class="card-head">
          <h2>Context propagation <span class="count">{report.contextFindings?.length || 0}</span></h2>
          <p class="what">
            What this checks: the greps the course calls "the cheapest thing to review" — <code>context.Background()</code>
            or <code>context.TODO()</code> used outside <code>main.go</code>, a <code>context.Context</code> stored as a
            struct field (lifetime mismatch), a <code>ctx</code> parameter that isn't first in the function signature,
            and HTTP calls (<code>http.Get</code>/<code>http.Post</code>/<code>http.DefaultClient</code>) with no explicit
            timeout or context. Matches inside string literals (e.g. a log message that mentions the pattern) are excluded.
          </p>
        </div>
        {#if !report.contextFindings?.length}
          <div class="muted">No context.Background()/TODO(), struct-stored contexts, misplaced ctx params, or unbounded HTTP calls found.</div>
        {:else}
          {#each report.contextFindings as c}
            <div class="finding">
              <div class="finding-msg">{c.kind} <span class="badge">{c.file}:{c.line}</span></div>
              <div class="finding-detail"><code>{c.snippet}</code></div>
            </div>
          {/each}
        {/if}
      </section>

      <!-- Dependency review (episode 08) -->
      <section class="card">
        <div class="card-head">
          <h2>Dependency review <span class="count">{report.dependencies?.findings?.length || 0}</span></h2>
          <p class="what">
            What this checks: <code>go list -m all</code> for every module in the build list — direct vs. indirect,
            pseudo-versions (pinned to a commit no release ever contained), and <b>indirect-mismatch</b> (a module
            go.mod marks indirect that internal code actually imports directly — usually means go.mod is stale and
            needs <code>go mod tidy</code>). Removal cost is the fraction of internal files that import each module,
            as a rough proxy for how entangled it is. CGO is found by walking the full transitive dependency closure
            (<code>go list -deps ./...</code>) for any package with <code>CgoFiles</code>. <code>govulncheck</code> runs
            only if it's already installed on this machine — this tool never installs anything.
          </p>
        </div>

        {#if report.dependencies?.findings?.length}
          {#each report.dependencies.findings as f}
            <div class="finding">
              <span class="tag" class:landfill={f.category === 'indirect-mismatch'} class:inverted={f.category === 'replace'}>{f.category}</span>
              <div class="finding-msg">{f.message}</div>
              <div class="finding-detail">{f.detail}</div>
            </div>
          {/each}
        {:else}
          <div class="muted">No indirect-mismatches, pseudo-versions, replace, or exclude directives found.</div>
        {/if}

        <div class="subhead">CGO</div>
        <div class="row muted">
          {#if report.dependencies?.cgoPackages?.length}
            {report.dependencies.cgoPackages.length} package(s) in the transitive closure carry CGO files: {report.dependencies.cgoPackages.join(', ')}.
            {#if report.build?.cgoFreeBuild?.ok}A <code>CGO_ENABLED=0</code> build still succeeds, so this likely isn't load-bearing.{/if}
          {:else}
            No package in the transitive closure carries CGO files.
          {/if}
        </div>

        <div class="subhead">govulncheck</div>
        {#if !report.dependencies?.govulncheckAvailable}
          <div class="row muted">Not installed on this machine — skipped. Install it to include reachability-based vulnerability findings here.</div>
        {:else}
          <pre class="output">{report.dependencies.vulnOutput || '(no output)'}</pre>
        {/if}

        <div class="subhead">Direct dependencies ({(report.dependencies?.modules || []).filter(m => m.directlyImported).length})</div>
        <table class="tbl">
          <thead><tr><th>module</th><th>version</th><th title="marked indirect in go.mod">go.mod says</th><th title="fraction of internal files importing this module">removal cost</th></tr></thead>
          <tbody>
            {#each (report.dependencies?.modules || []).filter(m => m.directlyImported) as m}
              <tr>
                <td><code class="pkg">{m.path}</code></td>
                <td class="num">{m.version}</td>
                <td class="num">{m.indirect ? 'indirect' : 'direct'}</td>
                <td class="num">{m.removalCostFiles} file(s) ({(m.removalCostRatio * 100).toFixed(0)}%)</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </section>

      <!-- Build, release, reproducibility (episode 12) -->
      <section class="card">
        <div class="card-head">
          <h2>Build &amp; release</h2>
          <p class="what">
            What this checks: real commands run against this tree (output redirected to a scratch temp dir, nothing
            written into the workspace) — <code>go build ./...</code>, <code>go vet ./...</code>, and the same build
            again with <code>CGO_ENABLED=0</code>. Plus text checks: whether <code>-trimpath</code> appears in the
            Makefile/Dockerfile/CI workflows, whether go.mod pins a <code>toolchain</code>, whether the source uses
            <code>runtime/debug.ReadBuildInfo</code> vs. <code>-ldflags</code> for version info, and (if a Dockerfile
            exists) its final base image, <code>USER</code> directive, and whether <code>.dockerignore</code> excludes <code>.git</code>.
          </p>
        </div>

        <div class="kv-grid">
          <div class="kv"><span class="k">go build ./...</span><code class="v" class:bad={!report.build?.build?.ok}>{report.build?.build?.ok ? 'OK' : 'FAILED'}</code></div>
          <div class="kv"><span class="k">go vet ./...</span><code class="v" class:bad={!report.build?.vet?.ok}>{report.build?.vet?.ok ? 'OK' : 'FAILED'}</code></div>
          <div class="kv"><span class="k">CGO_ENABLED=0 build</span><code class="v" class:bad={!report.build?.cgoFreeBuild?.ok}>{report.build?.cgoFreeBuild?.ok ? 'OK' : 'FAILED'}</code></div>
        </div>

        {#if !report.build?.build?.ok || !report.build?.vet?.ok || !report.build?.cgoFreeBuild?.ok}
          {#each [['build', report.build?.build], ['vet', report.build?.vet], ['cgoFreeBuild', report.build?.cgoFreeBuild]] as [label, r]}
            {#if r && !r.ok}
              <div class="subhead warn">{label} output</div>
              <pre class="output">{r.output}</pre>
            {/if}
          {/each}
        {/if}

        <div class="subhead">Reproducibility &amp; version info</div>
        <div class="row"><span>-trimpath configured</span><span class="badge">{report.build?.trimpathConfigured ? report.build.trimpathFoundIn.join(', ') : 'not found'}</span></div>
        <div class="row"><span>toolchain pinned in go.mod</span><span class="badge">{report.build?.toolchainLine || 'not pinned'}</span></div>
        <div class="row"><span>version via runtime/debug.ReadBuildInfo</span><span class="badge">{report.build?.usesReadBuildInfo ? 'yes' : 'no'}</span></div>
        <div class="row"><span>version via -ldflags</span><span class="badge">{report.build?.usesLdflagsVersion ? 'yes' : 'no'}</span></div>

        <div class="subhead">Container</div>
        {#if !report.build?.hasDockerfile}
          <div class="muted">No Dockerfile found.</div>
        {:else}
          {#each report.build.dockerFindings as d}
            <div class="row unknown">{d}</div>
          {/each}
        {/if}
      </section>

      <!-- Automate everything mechanical (episode 14) -->
      <section class="card">
        <div class="card-head">
          <h2>Lint policy <span class="count">{report.lint?.issueCount || 0} issue(s)</span></h2>
          <p class="what">
            What this checks: Vidian is a read-only viewer, so this never writes a config to your repo. If a
            <code>.golangci.yml</code>/<code>.yaml</code>/<code>.toml</code>/<code>.json</code> already exists it's shown
            as-is — the course's "config as policy document." If none exists, a justified baseline is suggested as text
            to copy in yourself (each linter has a one-line reason, so the config stays reviewable). <code>gofmt -l</code>
            lists files that would change under formatting (never runs <code>-w</code>). <code>golangci-lint</code> itself
            only runs if it's already installed on this machine — against your existing config, or the suggested one via
            a temp file if you have none — and the CI workflows are checked for whether they invoke golangci-lint at all,
            and whether <code>--new-from-rev</code> is used to grandfather old issues instead of blocking on them.
          </p>
        </div>

        <div class="subhead">gofmt</div>
        {#if !report.lint?.gofmtFiles?.length}
          <div class="muted">Every file is already gofmt-formatted.</div>
        {:else}
          <div class="chip-row">
            {#each report.lint.gofmtFiles as f}
              <code class="chip">{f}</code>
            {/each}
          </div>
        {/if}

        <div class="subhead">golangci-lint</div>
        {#if !report.lint?.golangciLintAvailable}
          <div class="row muted">Not installed on this machine — skipped. Install it to run real lint findings here.</div>
        {:else if report.lint?.lintTimedOut}
          <div class="row muted">Timed out after 90s — try running it locally for full output.</div>
        {:else}
          <pre class="output" class:ok={report.lint.issueCount === 0}>{report.lint.lintOutput || '(no output)'}</pre>
        {/if}

        <div class="subhead">CI</div>
        <div class="row"><span>golangci-lint runs in CI</span><span class="badge">{report.lint?.ciRunsLint ? 'yes' : 'no'}</span></div>
        {#if report.lint?.ciRunsLint}
          <div class="row"><span>uses --new-from-rev (grandfathers old issues)</span><span class="badge">{report.lint?.ciUsesNewFromRev ? 'yes' : 'no'}</span></div>
        {/if}

        <div class="subhead">{report.lint?.configExists ? `Existing config (${report.lint.configPath})` : 'Suggested baseline (not written to disk)'}</div>
        <pre class="output config">{report.lint?.configExists ? report.lint.configContent : report.lint?.suggestedConfig}</pre>
      </section>

      <!-- Concurrency read as ownership (episode 06) -->
      <section class="card">
        <div class="card-head">
          <h2>Concurrency <span class="count">{report.concurrency?.findings?.length || 0}</span></h2>
          <p class="what">
            What this checks: only the ownership patterns the course itself calls mechanical — a
            <code>go</code> statement launched inside a <code>for</code>/<code>range</code> loop with no visible
            limiter (unbounded concurrency if the bound comes from external input); <code>time.After(...)</code>
            inside a <code>select</code> that's itself inside a loop (allocates a timer every iteration);
            <code>errgroup.WithContext(ctx)</code> whose derived context is assigned to <code>_</code> (cancellation
            becomes decorative); and <code>time.NewTicker</code> with no matching <code>.Stop()</code> anywhere in the
            same function. <b>Not checked</b>: channel-close ownership with multiple senders — verifying who's
            allowed to close needs real data-flow tracing, and a wrong verdict there is worse than no verdict, so it's
            left to a human reader.
          </p>
        </div>

        {#if !report.concurrency?.findings?.length}
          <div class="muted">No unbounded loop-goroutines, loop-scoped time.After, discarded errgroup contexts, or un-stopped tickers found.</div>
        {:else}
          {#each report.concurrency.findings as f}
            <div class="finding">
              <div class="finding-msg"><code>{f.kind}</code> <span class="badge">{f.file}:{f.line}</span></div>
              <div class="finding-detail"><code>{f.snippet}</code></div>
              <div class="finding-detail">{f.detail}</div>
            </div>
          {/each}
        {/if}

        <div class="subhead">CI</div>
        <div class="row"><span><code>go test -race</code> runs in CI</span><span class="badge">{report.concurrency?.raceInCI ? 'yes' : 'no'}</span></div>
      </section>
    {:else}
      <div class="state-msg">
        <button class="run-btn-inline" onclick={() => store.runGoReview()}>Run review</button>
      </div>
    {/if}
  </div>
</div>

<style>
  .gorev {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #121214;
    color: #e3e3e6;
    font-family: system-ui, sans-serif;
    box-sizing: border-box;
    overflow: hidden;
  }

  .gorev-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 20px 32px 16px;
    flex-wrap: wrap;
  }

  .title-row { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }

  h1 { font-size: 20px; font-weight: 700; color: #f0f0f4; margin: 0; }

  .tagline { font-size: 12px; color: #6b7280; }

  .run-btn, .run-btn-inline {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 12px;
    font-weight: 600;
    color: #a5b4fc;
    background: rgba(129, 140, 248, 0.12);
    border: 1px solid rgba(129, 140, 248, 0.25);
    border-radius: 6px;
    padding: 6px 14px;
    cursor: pointer;
  }
  .run-btn:hover:not(:disabled), .run-btn-inline:hover { background: rgba(129, 140, 248, 0.22); }
  .run-btn:disabled { opacity: 0.6; cursor: default; }

  .gorev-body {
    flex: 1;
    overflow-y: auto;
    padding: 0 32px 40px;
    max-width: 980px;
  }

  .state-msg { padding: 40px 8px; color: #6b7280; font-size: 13px; }
  .state-msg.error { color: #f87171; }

  .card {
    background: #1b1b20;
    border: 1px solid #2d2d34;
    border-radius: 10px;
    padding: 20px 22px;
    margin-bottom: 20px;
  }

  .card-head { margin-bottom: 14px; }

  h2 {
    font-size: 14.5px;
    font-weight: 700;
    color: #f0f0f4;
    margin: 0 0 6px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .count {
    font-size: 11px;
    font-weight: 600;
    color: #9ca3af;
    background: rgba(255, 255, 255, 0.06);
    border-radius: 10px;
    padding: 1px 9px;
  }

  .what {
    font-size: 12px;
    line-height: 1.65;
    color: #9095a1;
    margin: 0;
  }
  .what code {
    font-family: 'Fira Code', monospace;
    font-size: 11.5px;
    color: #c4c4cc;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 3px;
    padding: 0 4px;
  }
  .what b { color: #d4d4d8; }

  .kv-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 6px 24px;
    margin-bottom: 6px;
  }
  .kv { display: flex; gap: 6px; font-size: 12px; }
  .kv .k { color: #6b7280; }
  .kv .v { color: #e3e3e6; font-family: 'Fira Code', monospace; }

  .subhead {
    margin-top: 16px;
    margin-bottom: 6px;
    font-size: 10.5px;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: #6b7280;
    font-weight: 700;
  }
  .subhead.warn { color: #fbbf24; }

  .chip-row { display: flex; flex-wrap: wrap; gap: 6px; }
  .chip {
    font-family: 'Fira Code', monospace;
    font-size: 11.5px;
    color: #a5b4fc;
    background: rgba(129, 140, 248, 0.1);
    border-radius: 4px;
    padding: 3px 8px;
  }

  .tbl { width: 100%; border-collapse: collapse; font-size: 12px; }
  .tbl th {
    text-align: left;
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: #6b7280;
    font-weight: 600;
    padding: 4px 8px;
    border-bottom: 1px solid #2d2d34;
  }
  .tbl td { padding: 4px 8px; border-bottom: 1px solid rgba(255,255,255,0.03); }
  .tbl td.num { text-align: right; color: #9ca3af; font-variant-numeric: tabular-nums; width: 70px; }
  .tbl .pkg { font-family: 'Fira Code', monospace; color: #a5b4fc; font-size: 11.5px; }

  .two-col { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; margin-top: 4px; }

  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
    padding: 2px 0;
    font-size: 12px;
  }
  .row code { font-family: 'Fira Code', monospace; font-size: 11.5px; color: #c4c4cc; }
  .row.unknown { color: #fbbf24; font-size: 11.5px; display: block; }

  .badge {
    flex-shrink: 0;
    font-size: 10px;
    color: #9ca3af;
    background: rgba(255, 255, 255, 0.06);
    border-radius: 3px;
    padding: 0 5px;
  }

  .muted { color: #6b7280; font-size: 12px; }
  .muted code { font-family: 'Fira Code', monospace; }

  .kv .v.bad { color: #f87171; }

  .output {
    font-family: 'Fira Code', monospace;
    font-size: 11px;
    color: #f87171;
    background: rgba(248, 113, 113, 0.08);
    border: 1px solid rgba(248, 113, 113, 0.2);
    border-radius: 6px;
    padding: 10px 12px;
    margin: 4px 0 0;
    max-height: 220px;
    overflow: auto;
    white-space: pre-wrap;
    word-break: break-word;
  }
  .output.ok {
    color: #a3e635;
    background: rgba(163, 230, 53, 0.06);
    border-color: rgba(163, 230, 53, 0.18);
  }
  .output.config {
    color: #c4c4cc;
    background: rgba(255, 255, 255, 0.03);
    border-color: #2d2d34;
    max-height: 260px;
  }

  .finding {
    padding: 10px 0;
    border-top: 1px solid rgba(255, 255, 255, 0.05);
  }
  .finding:first-of-type { border-top: none; padding-top: 0; }
  .finding.flagged { border-left: 2px solid #f59e0b; padding-left: 10px; margin-left: -2px; }

  .tag {
    display: inline-block;
    font-size: 9.5px;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    padding: 1px 6px;
    border-radius: 3px;
    background: rgba(255, 255, 255, 0.08);
    color: #9ca3af;
    margin-bottom: 4px;
  }
  .tag.landfill { background: rgba(251, 191, 36, 0.15); color: #fbbf24; }
  .tag.inverted { background: rgba(248, 113, 113, 0.15); color: #f87171; }

  .finding-msg { font-size: 13px; color: #e3e3e6; line-height: 1.5; }
  .finding-msg code { font-family: 'Fira Code', monospace; color: #a5b4fc; }
  .finding-detail { font-size: 11.5px; color: #8e8e93; line-height: 1.55; margin-top: 3px; }
  .finding-detail code { font-family: 'Fira Code', monospace; color: #c4c4cc; }
  .flag-reason { color: #fbbf24; }

  @media (max-width: 720px) {
    .two-col { grid-template-columns: 1fr; gap: 8px; }
    .gorev-header { padding: 16px 16px 12px; }
    .gorev-body { padding: 0 16px 32px; }
  }
</style>
