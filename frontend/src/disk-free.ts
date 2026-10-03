export type DiskFilesystem = {source:string;type:string;total:string;used:string;available:string;percent:number|null;mount:string;virtual:boolean};
export type DiskSnapshot = {filesystems:DiskFilesystem[]|null;updated_at:number;stale:boolean;message:string};
export function diskSize(value:string):string {
    if(value==='-')return 'Not available';
    const match=value.match(/^(-?[\d.]+)([KMGTPEZYRQ])$/);
    return match ? `${match[1]} ${match[2]==='K' ? 'KiB' : match[2]+'iB'}` : `${value} B`;
}
export function createDiskView(read:()=>Promise<DiskSnapshot>){
    const root=document.getElementById('disk-view')!;
    const list=document.getElementById('disk-volumes')!;
    const extras=document.getElementById('disk-extras') as HTMLDetailsElement;
    const extraList=document.getElementById('disk-system-volumes')!;
    const summary=document.getElementById('disk-extra-summary')!;
    const status=document.getElementById('disk-status')!;
    const button=document.getElementById('disk-refresh') as HTMLButtonElement;
    let active=false,generation=0,timer:ReturnType<typeof setTimeout>,pending:Promise<DiskSnapshot>|undefined,signature='';
    function card(d:DiskFilesystem){
        const card=document.createElement('article');card.className='disk-card';
        const heading=document.createElement('div');heading.className='disk-heading';
        const title=document.createElement('h3');title.textContent=d.mount==='/' ? 'System · /' : d.mount;
        const percent=document.createElement('span');percent.textContent=d.percent==null ? '—' : `${d.percent}% used`;
        heading.append(title,percent);card.append(heading);
        const source=document.createElement('p');source.className='disk-source';source.textContent=`${d.source} · ${d.type}`;card.append(source);
        const free=document.createElement('div');free.className='disk-free-value';
        const amount=document.createElement('strong');amount.textContent=diskSize(d.available);
        const label=document.createElement('span');label.textContent='available';free.append(amount,label);card.append(free);
        if(d.percent!=null){const bar=document.createElement('progress');bar.max=100;bar.value=Math.min(100,d.percent);bar.setAttribute('aria-label',`${d.mount}: ${d.percent}% used`);card.append(bar);}
        const sizes=document.createElement('div');sizes.className='disk-sizes';
        for(const [label,value] of [['Used',d.used],['Total',d.total]]){const cell=document.createElement('span');cell.textContent=`${label} ${diskSize(value)}`;sizes.append(cell);}card.append(sizes);
        if(!d.virtual && d.percent!=null && d.percent>=85){
            const note=document.createElement('p');note.className='disk-warning';
            note.textContent=d.percent>=100 ? 'Full · check available space' : d.percent>=95 ? 'Very little space remaining' : 'Storage nearly full';card.append(note);
        }
        return card;
    }
    function render(s:DiskSnapshot){
        const rows=s.filesystems || [];
        const next=JSON.stringify(rows);
        if(next!==signature){
            signature=next;
            const scroll=document.getElementById('disk-scroll')!;const top=scroll.scrollTop;
            list.replaceChildren(...rows.filter(d=>!d.virtual).map(card));
            const virtual=rows.filter(d=>d.virtual);
            extraList.replaceChildren(...virtual.map(card));extras.hidden=virtual.length===0;
            summary.textContent=`Memory & system (${virtual.length})`;
            scroll.scrollTop=top;
        }
        status.textContent=[s.stale ? 'Last known readings' : '',s.message,s.updated_at ? `Updated ${new Date(s.updated_at*1000).toLocaleTimeString()}` : ''].filter(Boolean).join(' · ');
        root.classList.toggle('is-stale',s.stale);
    }
    async function refresh(){
        if(!active)return;
        clearTimeout(timer);const token=generation;button.disabled=true;
        if(!pending)pending=read().finally(()=>{pending=undefined;});
        try{const s=await pending;if(active && token===generation)render(s);}
        catch{if(active && token===generation){root.classList.add('is-stale');status.textContent='Cannot refresh disk space. Displayed readings may be outdated.';}}
        finally{if(active && token===generation){button.disabled=false;clearTimeout(timer);timer=setTimeout(()=>{void refresh();},5000);}}
    }
    button.onclick=()=>{void refresh();};
    return {
        show(){active=true;generation++;root.hidden=false;status.textContent='Refreshing disk space…';void refresh();},
        close(){active=false;generation++;clearTimeout(timer);root.hidden=true;},
        refresh,
    };
}
