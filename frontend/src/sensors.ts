export type SensorSnapshot = {
    readings: {chip:string; label:string; kind:string; unit:string; value:number|null; high?:number; critical?:number}[] | null;
    updated_at:number; stale:boolean; message:string;
};

// Poll only while the detail is visible. A generation token discards late reads
// after changing candidates, opening Settings, or hiding the native window.
export function createSensorsView(read:() => Promise<SensorSnapshot>) {
    const root = document.getElementById('sensors-view')!;
    const list = document.getElementById('sensors-readings')!;
    const status = document.getElementById('sensors-status')!;
    const button = document.getElementById('sensors-refresh') as HTMLButtonElement;
    let active = false, generation = 0, timer:ReturnType<typeof setTimeout>;
    let pending:Promise<SensorSnapshot>|undefined;
    let previousRows = '';
    function render(data:SensorSnapshot) {
        const readings = data.readings || [];
        // Avoid rebuilding the scrollable list when values haven't changed.
        const signature = JSON.stringify(readings);
        if (signature !== previousRows) {
            previousRows = signature;
            const fragment = document.createDocumentFragment();
            let chip = '', group:HTMLDListElement|undefined;
            for (const r of readings) {
                if (!group || r.chip !== chip) {
                    chip = r.chip;
                    const heading = document.createElement('h3'); heading.textContent = chip;
                    fragment.append(heading);
                    group = document.createElement('dl'); fragment.append(group);
                }
                const row = document.createElement('div');
                const label = document.createElement('dt'); label.textContent = r.label;
                const value = document.createElement('dd');
                value.textContent = r.value == null ? 'Not available' : `${new Intl.NumberFormat(undefined,{maximumFractionDigits:r.kind === 'fan' ? 0 : r.kind === 'temp' ? 1 : 3}).format(r.value)} ${r.unit}`;
                row.append(label,value);
                if (r.high != null || r.critical != null) {
                    const limits = document.createElement('small');
                    limits.textContent = [r.high != null ? `High ${r.high} °C` : '',r.critical != null ? `Critical ${r.critical} °C` : ''].filter(Boolean).join(' · ');
                    row.append(limits);
                }
                group.append(row);
            }
            const scroll = list.scrollTop;
            list.replaceChildren(fragment); list.scrollTop = scroll;
        }
        const updated = data.updated_at ? `Updated ${new Date(data.updated_at*1000).toLocaleTimeString()}` : '';
        status.textContent = [data.stale ? 'Last known readings' : '',data.message,updated].filter(Boolean).join(' · ');
        root.classList.toggle('is-stale',data.stale);
    }
    async function refresh() {
        if (!active) return;
        clearTimeout(timer);
        const token = generation;
        button.disabled = true;
        if (!pending) pending = read().finally(() => {pending = undefined;});
        try {
            const data = await pending;
            if (active && token === generation) render(data);
        } catch {
            if (active && token === generation) {
                root.classList.add('is-stale');
                status.textContent = 'Cannot refresh sensors. Displayed readings may be outdated.';
            }
        } finally {
            if (active && token === generation) {
                button.disabled = false;
                clearTimeout(timer);
                timer = setTimeout(() => {void refresh();},2000);
            }
        }
    }
    button.onclick = () => {void refresh();};
    return {
        show() { active = true; generation++; root.hidden = false; status.textContent = 'Refreshing sensors…'; void refresh(); },
        close() { active = false; generation++; clearTimeout(timer); root.hidden = true; },
        refresh,
    };
}
