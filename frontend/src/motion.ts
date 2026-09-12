const reduceMotion = () => matchMedia('(prefers-reduced-motion: reduce)').matches;
const easing = 'cubic-bezier(.22, 1, .36, 1)';
const running = new WeakMap<HTMLElement, Animation>();
function play(el: HTMLElement, frames: Keyframe[], duration: number) {
    running.get(el)?.cancel();
    if (reduceMotion() || !el.animate) return;
    const animation = el.animate(frames, { duration, easing });
    running.set(el, animation);
}
// Animate painted layers, never layout, blur or the search input's scale.
export function reveal() {
    const search = document.querySelector<HTMLElement>('.search-area')!;
    const workspace = document.querySelector<HTMLElement>('.workspace')!;
    play(search, [{ opacity: 0 }, { opacity: 1 }], 260);
    play(workspace, [{ opacity: 0 }, { opacity: 1 }], 300);
}
export function selectIcon(el: HTMLElement) {
    play(el, [{ transform: 'scale(1)' }, { transform: 'scale(1.12)', offset: .3 }, { transform: 'scale(1)' }], 230);
}
export function revealDetails(el: HTMLElement) {
    play(el, [{ opacity: .85 }, { opacity: 1 }], 150);
}
