export type ProcessSnapshot = {
    cpu:number|null; cpus:number; memory_used:number; memory_total:number; swap_used:number; swap_total:number;
    process_count:number; skipped:number; processes:{pid:number; name:string; cpu:number|null; memory:number}[]|null;
    updated_at:number; stale:boolean; message:string;
};
export function isHtop(item:{kind:string;id:string;path:string}|undefined):boolean {
    return item?.kind === 'app' && (item.id === 'app:htop.desktop' || item.path.split('/').pop() === 'htop.desktop');
}
const bytes = (n:number) => n >= 1024**3 ? `${(n/1024**3).toFixed(1)} GiB` : `${(n/1024**2).toFixed(0)} MiB`;
const percent = (n:number|null) => n == null ? '—' : `${n.toFixed(1)}%`;
export function createProcessView(read:() => Promise<ProcessSnapshot>) {
    const root=document.getElementById('process-view')!;
    const status=document.getElementById('process-status')!;
    const summary=document.getElementById('process-summary')!;
    const list=document.getElementById('process-list')!;
    const caption=document.getElementById('process-caption')!;
    const button=document.getElementById('process-refresh') as HTMLButtonElement;
    let active=false,generation=0,timer:ReturnType<typeof setTimeout>,pending:Promise<ProcessSnapshot>|undefined;
    function render(s:ProcessSnapshot) {
        const fragment=document.createDocumentFragment();
        for(const [label,value,ratio] of [
            ['CPU',`${percent(s.cpu)} · ${s.cpus} threads`,s.cpu],
            ['RAM',`${bytes(s.memory_used)} / ${bytes(s.memory_total)}`,s.memory_total ? s.memory_used/s.memory_total*100 : null],
            ['Swap',s.swap_total ? `${bytes(s.swap_used)} / ${bytes(s.swap_total)}` : 'Not configured',s.swap_total ? s.swap_used/s.swap_total*100 : null],
        ] as const) {
            const row=document.createElement('div');
            const title=document.createElement('span');title.textContent=label;
            const text=document.createElement('strong');text.textContent=s.updated_at ? value : '—';
            row.append(title,text);
            if(ratio!=null && s.updated_at){const bar=document.createElement('progress');bar.max=100;bar.value=ratio;bar.setAttribute('aria-label',`${label} usage`);row.append(bar);}
            fragment.append(row);
        }
        summary.replaceChildren(fragment);
        caption.textContent=s.updated_at ? `TOP CPU · ${s.process_count} processes${s.skipped ? ` · ${s.skipped} unavailable` : ''}` : 'TOP CPU';
        const rows=document.createDocumentFragment();
        for(const p of s.processes || []){
            const row=document.createElement('tr');
            for(const value of [p.name,String(p.pid),percent(p.cpu),bytes(p.memory)]){
                const cell=document.createElement('td');cell.textContent=value;cell.title=value;row.append(cell);
            }
            rows.append(row);
        }
        list.replaceChildren(rows);
        status.textContent=[s.stale ? 'Last known readings' : '',s.message,s.updated_at ? `Updated ${new Date(s.updated_at*1000).toLocaleTimeString()}` : ''].filter(Boolean).join(' · ');
        root.classList.toggle('is-stale',s.stale);
    }
    async function refresh(){
        if(!active)return;
        clearTimeout(timer);const token=generation;button.disabled=true;
        if(!pending)pending=read().finally(()=>{pending=undefined;});
        try{const s=await pending;if(active && token===generation)render(s);}
        catch{if(active && token===generation){root.classList.add('is-stale');status.textContent='Cannot refresh processes. Displayed readings may be outdated.';}}
        finally{if(active && token===generation){button.disabled=false;clearTimeout(timer);timer=setTimeout(()=>{void refresh();},2000);}}
    }
    button.onclick=()=>{void refresh();};
    return {
        show(){active=true;generation++;root.hidden=false;status.textContent='Refreshing processes…';void refresh();},
        close(){active=false;generation++;clearTimeout(timer);root.hidden=true;},
    };
}
