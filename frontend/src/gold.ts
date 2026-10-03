export type GoldSnapshot={quotes:{code:string;name:string;buy:number;sell:number;buy_change:number|null;sell_change:number|null}[]|null;history:{date:string;buy:number;sell:number}[]|null;source_at:number;fetched_at:number;is_today:boolean;stale:boolean;message:string;chart_message:string};
const million=(n:number)=>new Intl.NumberFormat('en-GB',{minimumFractionDigits:2,maximumFractionDigits:2}).format(n/1e6);
const day=(iso:string)=>new Intl.DateTimeFormat('en-GB',{timeZone:'UTC',day:'2-digit',month:'short'}).format(new Date(`${iso}T00:00:00Z`));
const changed=(n:number|null)=>n==null?'No previous quote':`${n>0?'↑ +':n<0?'↓ −':'→ '}${million(Math.abs(n))} vs previous day`;
const stamp=(n:number)=>new Intl.DateTimeFormat('en-GB',{timeZone:'Asia/Ho_Chi_Minh',day:'2-digit',month:'short',year:'numeric',hour:'2-digit',minute:'2-digit',hour12:false}).format(n*1000);
export function createGoldView(read:(refresh:boolean)=>Promise<GoldSnapshot>,openSource:()=>Promise<void>){
 const $=(id:string)=>document.getElementById(id)!;
 const root=$('gold-view'),status=$('gold-status'),prices=$('gold-prices'),chart=$('gold-chart'),rows=$('gold-rows');
 const button=$('gold-refresh') as HTMLButtonElement,range=$('gold-day') as HTMLInputElement;
 let active=false,generation=0,timer:ReturnType<typeof setTimeout>,pending:Promise<GoldSnapshot>|undefined;
 let points:NonNullable<GoldSnapshot['history']>=[],chosenDate='',lastSignature='';
 function svgNode(name:string,attrs:Record<string,string|number>,text?:string){const n=document.createElementNS('http://www.w3.org/2000/svg',name);for(const [key,v]of Object.entries(attrs))n.setAttribute(key,String(v));if(text)n.textContent=text;return n;}
 function draw(){
  chart.replaceChildren();if(points.length<2)return;
  const selected=Math.max(0,points.findIndex(p=>p.date===chosenDate));
  const vals=points.flatMap(p=>[p.buy,p.sell]);const lo=Math.min(...vals),hi=Math.max(...vals),pad=Math.max((hi-lo)*.12,100000);
  const min=lo-pad,max=hi+pad;
  const x=(i:number)=>43+i/(points.length-1)*257,y=(v:number)=>12+(max-v)/(max-min)*88;
  const svg=svgNode('svg',{viewBox:'0 0 308 122',role:'img','aria-label':'SJC buy and sell prices, million VND per tael'});
  for(const v of [max,(max+min)/2,min]){svg.append(svgNode('line',{x1:43,x2:300,y1:y(v),y2:y(v),class:'gold-grid'}),svgNode('text',{x:38,y:y(v)+3,'text-anchor':'end',class:'gold-axis'},million(v)));}
  for(const [key,style] of [['buy','gold-buy-line'],['sell','gold-sell-line']] as const){
   const d=points.map((p,i)=>`${i?'L':'M'}${x(i).toFixed(2)},${y(p[key]).toFixed(2)}`).join(' ');
   svg.append(svgNode('path',{d,class:style,fill:'none'}));
  }
  svg.append(svgNode('text',{x:43,y:118,class:'gold-axis'},day(points[0].date)),svgNode('text',{x:300,y:118,'text-anchor':'end',class:'gold-axis'},day(points.at(-1)!.date)));
  const p=points[selected];svg.append(svgNode('line',{x1:x(selected),x2:x(selected),y1:12,y2:100,class:'gold-cursor'}));
  for(const key of ['buy','sell'] as const)svg.append(svgNode('circle',{cx:x(selected),cy:y(p[key]),r:3,class:'gold-dot'}));
  svg.addEventListener('pointermove',e=>{const event=e as PointerEvent;const rect=svg.getBoundingClientRect();const px=(event.clientX-rect.left)/rect.width*308;choose(Math.round(Math.max(0,Math.min(1,(px-43)/257))*(points.length-1)));});
  chart.append(svg);
  $('gold-point').textContent=`${day(p.date)} · Buy ${million(p.buy)} · Sell ${million(p.sell)}`;
 }
 function choose(i:number){const p=points[i];if(!p || chosenDate===p.date)return;chosenDate=p.date;range.value=String(i);draw();}
 range.oninput=()=>choose(Number(range.value));
 function render(s:GoldSnapshot){
  const q=s.quotes?.find(q=>q.code==='sjc');
  prices.replaceChildren();
  if(q){for(const [label,value,delta]of [['Buy',q.buy,q.buy_change],['Sell',q.sell,q.sell_change]] as const){
   const cell=document.createElement('div'),name=document.createElement('span'),amount=document.createElement('strong'),change=document.createElement('small');
   name.textContent=label;amount.textContent=million(value);change.textContent=changed(delta);cell.append(name,amount,change);prices.append(cell);
  }}
  $('gold-spread').textContent=q?`SJC · million VND/tael · Spread ${million(q.sell-q.buy)}`:'';
  const signature=JSON.stringify(s.history);
  if(signature!==lastSignature){lastSignature=signature;points=s.history||[];if(!points.some(p=>p.date===chosenDate))chosenDate=points.at(-1)?.date||'';
   range.max=String(Math.max(0,points.length-1));range.value=String(Math.max(0,points.findIndex(p=>p.date===chosenDate)));draw();}
  $('gold-chart-section').hidden=points.length<2;
  $('gold-chart-error').textContent=s.chart_message;
  rows.replaceChildren();
  for(const q of s.quotes||[]){const tr=document.createElement('tr');for(const text of [q.name,million(q.buy),million(q.sell)]){const td=document.createElement('td');td.textContent=text;tr.append(td);}rows.append(tr);}
  $('gold-table').hidden=!s.quotes?.length;
  const old=s.source_at && !s.is_today?'Today’s prices are not available from the source yet. ':'';
  status.textContent=[s.stale?'Cached prices.':'',old,s.message,s.source_at?`24h updated: ${stamp(s.source_at)}`:''].filter(Boolean).join(' ');
  root.classList.toggle('is-stale',s.stale || Boolean(old));
  $('gold-fetched').textContent=s.fetched_at?`Fetched: ${stamp(s.fetched_at)} · Vietnam time`:'';
 }
 async function refresh(force=false){
  if(!active)return;clearTimeout(timer);const token=generation;button.disabled=true;
  if(!pending)pending=read(force).finally(()=>{pending=undefined;});
  try{const s=await pending;if(active&&token===generation)render(s);}
  catch{if(active&&token===generation){root.classList.add('is-stale');status.textContent='Unable to refresh. Displayed prices may be outdated.';}}
  finally{if(active&&token===generation){button.disabled=false;clearTimeout(timer);timer=setTimeout(()=>{void refresh();},60000);}}
 }
 button.onclick=()=>{void refresh(true);};
 $('gold-source').onclick=()=>{void openSource().catch(()=>{status.textContent='Unable to open the browser.';});};
 return {show(){active=true;generation++;root.hidden=false;status.textContent='Loading prices from 24h…';void refresh();},close(){active=false;generation++;clearTimeout(timer);root.hidden=true;},refresh:()=>refresh(true)};
}
