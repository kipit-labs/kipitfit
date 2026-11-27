// The rest timer's end: a beep and a buzz, in the foreground only (Web Push in the background comes in P8).
let ctx: AudioContext | undefined;

/** Browsers only play sound after a tap: call this from one. */
export function unlockAudio() {
  try {
    ctx ??= new AudioContext();
    void ctx.resume();
  } catch {
    // No Web Audio: the vibration still runs.
  }
}

export function restOver() {
  navigator.vibrate?.([200, 100, 200]);
  if (!ctx) return;
  // Three short beeps.
  for (let i = 0; i < 3; i++) {
    const at = ctx.currentTime + i * 0.25;
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.frequency.value = 880;
    gain.gain.setValueAtTime(0.3, at);
    gain.gain.exponentialRampToValueAtTime(0.001, at + 0.18);
    osc.connect(gain).connect(ctx.destination);
    osc.start(at);
    osc.stop(at + 0.2);
  }
}
