<script>
  /**
   * Component Showcase
   * Visual test page for ButlerCharacter and KintsugiError
   */
  import ButlerCharacter from './ButlerCharacter.svelte';
  import KintsugiError from './KintsugiError.svelte';

  let butlerSentiment = $state('calm');
  let butlerLookDirection = 'center';
  let butlerMessage = $state('Cash runway is healthy. All systems operating normally.');
  let showError = $state(false);

  const messages = {
    calm: 'Cash runway is healthy. All systems operating normally.',
    neutral: 'No urgent actions required. Monitor the Opportunity Mandala.',
    urgent: 'Cash below 2 months! Consider closing Grade A deals immediately.'
  };

  function changeSentiment(sentiment) {
    butlerSentiment = sentiment;
    butlerMessage = messages[sentiment];
  }

  function changeLookDirection(direction) {
    butlerLookDirection = direction;
  }

  function triggerError() {
    showError = true;
    setTimeout(() => {
      showError = false;
    }, 5000);
  }
</script>

<div class="showcase-container" style="padding: var(--space-2xl); min-height: 100vh;">
  <h1 class="text-3xl font-serif" style="margin-bottom: var(--space-xl)">
    Component Showcase - Commandments VI & VII
  </h1>

  <!-- Butler Character Section -->
  <section style="margin-bottom: var(--space-2xl);">
    <h2 class="text-xl font-mono" style="margin-bottom: var(--space-md)">
      ButlerCharacter - The Living Assistant
    </h2>

    <div style="margin-bottom: var(--space-lg);">
      <ButlerCharacter
        sentiment={butlerSentiment}
        message={butlerMessage}
      />
    </div>

    <!-- Controls -->
    <div class="controls" style="display: flex; gap: var(--space-md); margin-bottom: var(--space-md);">
      <div>
        <div class="text-xs font-mono" style="margin-bottom: var(--space-xs)">Sentiment:</div>
        <button onclick={() => changeSentiment('calm')} class="btn">Calm</button>
        <button onclick={() => changeSentiment('neutral')} class="btn">Neutral</button>
        <button onclick={() => changeSentiment('urgent')} class="btn">Urgent</button>
      </div>

      <div>
        <div class="text-xs font-mono" style="margin-bottom: var(--space-xs)">Look Direction:</div>
        <button onclick={() => changeLookDirection('left')} class="btn">Left</button>
        <button onclick={() => changeLookDirection('center')} class="btn">Center</button>
        <button onclick={() => changeLookDirection('right')} class="btn">Right</button>
        <button onclick={() => changeLookDirection('up')} class="btn">Up</button>
        <button onclick={() => changeLookDirection('down')} class="btn">Down</button>
      </div>
    </div>

    <div class="feature-list">
      <div class="text-sm font-mono" style="margin-bottom: var(--space-xs)">Features:</div>
      <ul class="text-sm" style="list-style: disc; margin-left: var(--space-lg);">
        <li>Breathing animation (0.5 Hz calm, 0.8 Hz urgent)</li>
        <li>Eye movement tracking</li>
        <li>Subtle bow on mount</li>
        <li>Mood-based glow (green = calm, red = urgent)</li>
        <li>TextBloom message reveal</li>
      </ul>
    </div>
  </section>

  <!-- Kintsugi Error Section -->
  <section style="margin-bottom: var(--space-2xl);">
    <h2 class="text-xl font-mono" style="margin-bottom: var(--space-md)">
      KintsugiError - Beautiful Failures
    </h2>

    <div style="margin-bottom: var(--space-lg);">
      <button onclick={triggerError} class="btn-primary">
        Trigger Error (Watch the Gold Flow!)
      </button>
    </div>

    {#if showError}
      <KintsugiError
        message="Something went wrong, but now it's more beautiful than before. This is the kintsugi philosophy in action."
        show={true}
      />
    {/if}

    <div class="feature-list" style="margin-top: var(--space-lg);">
      <div class="text-sm font-mono" style="margin-bottom: var(--space-xs)">Features:</div>
      <ul class="text-sm" style="list-style: disc; margin-left: var(--space-lg);">
        <li>Gold crack appears (377ms Fibonacci timing)</li>
        <li>Automatic repair sequence (500ms delay)</li>
        <li>Gold flows into crack (800ms animation)</li>
        <li>Error message turns gold when repaired</li>
        <li>Svelte fade transitions</li>
      </ul>
    </div>
  </section>

  <!-- Philosophy Section -->
  <section>
    <h2 class="text-xl font-mono" style="margin-bottom: var(--space-md)">
      Steve Jobs' Vision
    </h2>

    <div class="quote-box" style="background: rgba(251, 191, 36, 0.1); padding: var(--space-lg); border-left: 4px solid var(--color-gold);">
      <p class="font-serif" style="margin-bottom: var(--space-md); font-style: italic;">
        "The Butler isn't a text box with emojis. The Butler is a CHARACTER. When you open the app, the Butler subtly bows. When it has insight, its eyes move to the relevant section. Text doesn't appear - it BLOOMS. The Butler BREATHES at 0.5 Hz."
      </p>
      <p class="font-serif" style="font-style: italic;">
        "When something goes wrong, you see a CRACK appear. A fine line, like a fracture in porcelain. Then gold flows into the crack. Kintsugi. Errors aren't failures. They're opportunities for beauty."
      </p>
    </div>
  </section>
</div>

<style>
  .btn {
    padding: var(--space-xs) var(--space-sm);
    background: var(--color-paper);
    border: 1px solid var(--color-ink-light);
    font-family: var(--font-mono);
    font-size: 12px;
    cursor: pointer;
    transition: all var(--duration-fast) var(--ease-wabi-sabi);
  }

  .btn:hover {
    background: var(--color-ink);
    color: var(--color-paper);
  }

  .btn-primary {
    padding: var(--space-sm) var(--space-md);
    background: var(--color-gold);
    color: var(--color-ink);
    border: none;
    font-family: var(--font-mono);
    font-size: 14px;
    cursor: pointer;
    transition: all var(--duration-fast) var(--ease-wabi-sabi);
  }

  .btn-primary:hover {
    background: var(--color-ink);
    color: var(--color-gold);
  }
</style>
