const reduceMotion = () => matchMedia('(prefers-reduced-motion: reduce)').matches;
const easing = 'cubic-bezier(.22, 1, .36, 1)';
const running = new WeakMap<HTMLElement, Animation>();
function play(el: HTMLElement, frames: Keyframe[], duration: number) {
    running.get(el)?.cancel();
    if (reduceMotion() || !el.animate) return;
    const animation = el.animate(frames, { duration, easing });
    running.set(el, animation);
}
// Expand upward from a slightly smaller, lowered surface. Capture both painted
// properties before cancelling so rapid reversals preserve the current pose.
async function transition(shell: HTMLElement, visible: boolean): Promise<boolean> {
    const current = getComputedStyle(shell);
    const from = Number(current.opacity);
    const fromTransform = current.transform;
    running.get(shell)?.cancel();
    shell.classList.toggle('is-armed', !visible);
    if (reduceMotion() || !shell.animate) return true;
    const to = visible ? 1 : 0;
    const toTransform = getComputedStyle(shell).transform;
    const duration = (visible ? 120 : 80) * Math.abs(to - from);
    const animation = shell.animate([
        { opacity: from, transform: fromTransform },
        { opacity: to, transform: toTransform },
    ], {
        duration, easing: visible ? 'cubic-bezier(.16,1,.3,1)' : 'cubic-bezier(.4,0,.8,.2)',
    });
    running.set(shell, animation);
    try { await animation.finished; return true; }
    catch { return false; } // Interrupted by another show/hide request.
    finally { if (running.get(shell) === animation) running.delete(shell); }
}
export function reveal(shell: HTMLElement) { return transition(shell, true); }
export function dismiss(shell: HTMLElement) { return transition(shell, false); }
export function conceal(shell: HTMLElement) {
    running.get(shell)?.cancel();
    running.delete(shell);
    shell.classList.add('is-armed');
}
export function selectIcon(el: HTMLElement) {
    play(el, [{ transform: 'scale(1)' }, { transform: 'scale(1.12)', offset: .3 }, { transform: 'scale(1)' }], 230);
}
export function revealDetails(el: HTMLElement) {
    play(el, [{ opacity: .85 }, { opacity: 1 }], 150);
}
