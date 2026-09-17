import test from 'node:test';
import assert from 'node:assert/strict';
import { reveal, dismiss, conceal } from '../src/motion.ts';

// Control animation completion to exercise interrupted native-window motion.
let reduced = false;
globalThis.matchMedia = () => ({ matches: reduced });
const collapsed = 'matrix(0.92, 0, 0, 0.92, 0, 12)';
globalThis.getComputedStyle = el => ({
    opacity: String(el.current ?? (el.armed ? 0 : 1)),
    transform: el.pose ?? (el.armed ? collapsed : 'none'),
});
function layer() {
    const el = { armed: true, current: null, animations: [] };
    el.classList = {
        toggle: (_name, armed) => { el.armed = armed; },
        add: () => { el.armed = true; },
    };
    el.animate = (frames, options) => {
        let finish, reject;
        const finished = new Promise((resolve, fail) => { finish = resolve; reject = fail; });
        const animation = { frames, options, finished, finish, cancel: () => {
            el.current = null; el.pose = null;
            reject(new Error('cancelled'));
        } };
        el.animations.push(animation);
        return animation;
    };
    return el;
}

test('rapid reversal preserves opacity and transform and cancels the old completion', async () => {
    const el = layer();
    const opening = reveal(el);
    assert.equal(el.animations[0].frames[0].transform, collapsed);
    assert.equal(el.animations[0].frames[1].transform, 'none');
    assert.equal(el.animations[0].options.duration, 120);
    el.current = .4;
    const midway = 'matrix(0.952, 0, 0, 0.952, 0, 7.2)';
    el.pose = midway;
    const closing = dismiss(el);
    assert.equal(await opening, false);
    assert.equal(el.animations[1].frames[0].opacity, .4);
    assert.equal(el.animations[1].frames[0].transform, midway);
    assert.equal(el.animations[1].frames[1].transform, collapsed);
    el.current = .2;
    const reopening = reveal(el);
    assert.equal(await closing, false);
    assert.equal(el.animations[2].frames[0].opacity, .2);
    assert.equal(el.armed, false);
    el.animations[2].finish();
    assert.equal(await reopening, true);
});

test('hide completes only after the animation, immediate handoff cancels it', async () => {
    const el = layer(); el.armed = false;
    let completed = false;
    const closing = dismiss(el).then(ok => { completed = ok; return ok; });
    await Promise.resolve();
    assert.equal(completed, false);
    assert.equal(el.animations[0].options.duration, 80);
    assert.equal(el.animations[0].frames[1].transform, collapsed);
    conceal(el);
    assert.equal(await closing, false);
    assert.equal(el.armed, true);
});

test('reduced motion changes visibility immediately without animations', async () => {
    reduced = true;
    try {
        const el = layer();
        assert.equal(await reveal(el), true);
        assert.equal(el.armed, false);
        assert.equal(await dismiss(el), true);
        assert.equal(el.armed, true);
        assert.equal(el.animations.length, 0);
    } finally { reduced = false; }
});
