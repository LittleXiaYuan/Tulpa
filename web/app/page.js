'use client';

import { useEffect, useId, useState } from 'react';

const assets = [
  ['BTC', 'Bitcoin', '₿', '#f3a94d'],
  ['ETH', 'Ethereum', 'Ξ', '#97a5ff'],
  ['SOL', 'Solana', '≋', '#b380ff'],
  ['XRP', 'XRP', '×', '#77daee'],
];
const intervals = [['15m','15m'], ['1H','1h'], ['4H','4h'], ['1Dutc','1d']];
const nav = [['overview', 'Overview', '▦'], ['strategy', 'Strategy lab', '⌁']];
const price = n => n.toLocaleString('en-US', { maximumFractionDigits: n < 1 ? 4 : 2, minimumFractionDigits: n < 1 ? 4 : 2 });
const compact = new Intl.NumberFormat('en-US', {notation:'compact', maximumFractionDigits:2});
const dateFormat = new Intl.DateTimeFormat('en-GB', {month:'short', day:'numeric', timeZone:'UTC'});
const timeFormat = new Intl.DateTimeFormat('en-GB', {timeStyle:'medium', timeZone:'UTC'});
const pct = n => (n > 0 ? '+' : '') + n.toFixed(2) + '%';
const tone = n => n >= 0 ? 'positive' : 'negative';

function PriceChart({ candles, bar }) {
  const [hover, setHover] = useState(null);
  const low = Math.min(...candles.map(c => c.low));
  const high = Math.max(...candles.map(c => c.high));
  const padding = Math.max((high-low)*.08, high*.0001);
  const min = low-padding, max = high+padding;
  const maxVolume = Math.max(1, ...candles.map(c => c.volume));
  const step = 820/candles.length;
  const x = i => 12+(i+.5)*step;
  const y = v => 13 + (max-v)/(max-min)*228;
  const index = hover ?? candles.length-1;
  const candle = candles[index];
  const label = value => value < 1 ? value.toFixed(4) : compact.format(value);
  const ticks = Array.from({length:5}, (_,i) => max-(max-min)*i/4);
  const timeline = Array.from({length:5}, (_,i) => Math.round((candles.length-1)*i/4));
  return <>
    <div className="ohlc" aria-live="off">
      <span>{dateFormat.format(candle.time)} · {timeFormat.format(candle.time)} UTC</span>
      <span>O <b>{price(candle.open)}</b></span><span>H <b>{price(candle.high)}</b></span>
      <span>L <b>{price(candle.low)}</b></span><span>C <b>{price(candle.close)}</b></span>
    </div>
    <svg className="price-chart" viewBox="0 0 905 320" role="img" aria-label="Candlestick price chart with quote volume" onPointerLeave={() => setHover(null)}
      onPointerMove={event => {
        const box = event.currentTarget.getBoundingClientRect();
        const position = (event.clientX-box.left)*905/box.width;
        setHover(Math.max(0, Math.min(candles.length-1, Math.floor((position-12)/step))));
      }}>
      {ticks.map(t => <g key={t}><path d={`M12 ${y(t)} H836`} className="gridline"/><text x="850" y={y(t)+4}>{label(t)}</text></g>)}
      {timeline.map((i,k) => <g key={i}><path d={`M${x(i)} 13 V289`} className="gridline"/>
        <text x={x(i)} y="313" textAnchor={k===0?'start':k===4?'end':'middle'}>{bar==='15m'?timeFormat.format(candles[i].time).slice(0,5):dateFormat.format(candles[i].time)}</text></g>)}
      {candles.map((c,i) => {
        const height = 42*c.volume/maxVolume;
        const width = step*.64;
        const up = c.close >= c.open;
        return <g key={c.time} className={up?'bull':'bear'}>
          <line x1={x(i)} x2={x(i)} y1={y(c.high)} y2={y(c.low)}/>
          <rect x={x(i)-width/2} y={Math.min(y(c.open),y(c.close))} width={width} height={Math.max(1,Math.abs(y(c.open)-y(c.close)))}/>
          <rect opacity=".28" x={x(i)-width/2} y={289-height} width={width} height={height}/>
        </g>;
      })}
      {hover !== null && <path d={`M${x(index)} 13 V289 M12 ${y(candle.close)} H836`} className="crosshair"/>}
    </svg>
  </>;
}

function Equity({ values }) {
  const id = useId().replaceAll(':','');
  const min = Math.min(1,...values), max = Math.max(1.001,...values);
  const points = values.map((v,i) => [i*320/(values.length-1), 68-(v-min)/(max-min)*61].join(',')).join(' ');
  return <svg className="equity" viewBox="0 0 320 75" role="img" aria-label="Simulated strategy equity curve">
    <defs><linearGradient id={id} x2="0" y2="1"><stop stopColor="#b7f96a" stopOpacity=".24"/><stop offset="100%" stopColor="#b7f96a" stopOpacity="0"/></linearGradient></defs>
    <polygon points={`0,75 ${points} 320,75`} fill={`url(#${id})`}/>
    <polyline points={points} fill="none" stroke="#b7f96a" strokeWidth="1.5"/>
  </svg>;
}

const defaultStrategy = { strategy: 'sma', fast: '12', slow: '26', feeBps: '10', slippageBps: '0' };
const fields = [
  ['fast', 'Fast period', 2, 99],
  ['slow', 'Slow period', 3, 100],
  ['feeBps', 'Fee per side (bps)', 0, 500],
  ['slippageBps', 'Slippage per side (bps)', 0, 500],
];

function StrategyLab({ config, onApply, result, feedError }) {
  const [draft, setDraft] = useState(config);
  const [validation, setValidation] = useState('');
  const dirty = Object.keys(config).some(key => draft[key] !== config[key]);

  function submit(event) {
    event.preventDefault();
    if (Number(draft.fast) >= Number(draft.slow)) {
      setValidation('Fast period must be smaller than slow period.');
      return;
    }
    setValidation('');
    onApply(draft);
  }

  function update(event) {
    const {name, value} = event.target;
    setDraft(current => ({...current, [name]:value}));
    setValidation('');
  }

  return <section className="card strategy" id="strategy" aria-labelledby="strategy-heading">
    <div className="panel-head">
      <div><h2 id="strategy-heading">Strategy lab</h2><p>{config.strategy.toUpperCase()} {config.fast} / {config.slow} · Long or cash</p></div>
      <span className="tag">BACKTEST</span>
    </div>
    <form className="strategy-form" onSubmit={submit}>
      <label>Strategy<select name="strategy" value={draft.strategy} onChange={update}>
        <option value="sma">SMA trend</option><option value="ema">EMA trend</option>
      </select></label>
      {fields.map(([name,label,min,max]) => <label key={name}>{label}
        <input name={name} type="number" min={min} max={max} step="1" required value={draft[name]} onChange={update}/>
      </label>)}
      <button type="submit" disabled={!dirty}>Run backtest</button>
      {validation && <p role="alert">{validation}</p>}
    </form>
    {result && !result.error ? <>
      <div className="strategy-content">
        <dl>{[
          ['Net return',pct(result.return),tone(result.return)],['Buy & hold',pct(result.benchmark),tone(result.benchmark)],
          ['Max drawdown',result.drawdown.toFixed(2)+'%',''],['Closed trades',result.trades,''],
        ].map(([label,value,color])=><div key={label}><dt>{label}</dt><dd className={color}>{value}</dd></div>)}</dl>
        <div className="curve"><span>STRATEGY EQUITY</span><Equity values={result.equity}/></div>
      </div>
      <div className="signal"><span>Latest confirmed signal</span><strong className={result.signal==='long'?'positive':''}>{result.signal==='long'?'Long':'Cash'}</strong>
        <time dateTime={new Date(result.signalTime).toISOString()}>{dateFormat.format(result.signalTime)} · {timeFormat.format(result.signalTime)} UTC</time>
      </div>
      <details className="trade-details">
        <summary>Trade history ({result.trades})</summary>
        {result.trades ? <div className="trade-scroll"><table>
          <thead><tr><th scope="col">Entry (UTC)</th><th scope="col">Exit (UTC)</th><th scope="col">Buy (USDT)</th><th scope="col">Sell (USDT)</th><th scope="col">Net return</th><th scope="col">Exit reason</th></tr></thead>
          <tbody>{[...result.ledger].reverse().map(trade => <tr key={trade.entryTime}>
            <td>{dateFormat.format(trade.entryTime)} {timeFormat.format(trade.entryTime).slice(0,5)}</td>
            <td>{dateFormat.format(trade.exitTime)} {timeFormat.format(trade.exitTime).slice(0,5)}</td>
            <td>{price(trade.entryPrice)}</td><td>{price(trade.exitPrice)}</td>
            <td className={tone(trade.return)}>{pct(trade.return)}</td><td>{trade.reason==='signal'?'Cash signal':'End of window'}</td>
          </tr>)}</tbody>
        </table></div> : <p>No trades in this window.</p>}
      </details>
    </> : <div className="strategy-empty" role="status">{result?.error || (feedError?'Backtest unavailable':'Calculating backtest…')}</div>}
    <p className="method">Signals use confirmed closes and execute at the next open. Both returns include {config.feeBps} bps in fees and {config.slippageBps} bps in slippage per side. Positions close at the end of the window.</p>
  </section>;
}

export default function Terminal() {
  const [asset, setAsset] = useState('BTC');
  const [bar, setBar] = useState('1H');
  const [mode, setMode] = useState('demo');
  const [refresh, setRefresh] = useState(0);
  const [config, setConfig] = useState(defaultStrategy);
  const run = JSON.stringify(config);
  const [state,setState] = useState({key:'', data:null, error:''});
  const key = asset + bar + mode;
  const active = state.key === key ? state : {data:null,error:''};
  const {data,error} = active;
  const selected = assets.find(a=>a[0]===asset);

  useEffect(() => {
    const controller = new AbortController();
    let timer;
    async function load() {
      let interval = 15000;
      try {
        const params = new URLSearchParams({instrument:asset+'-USDT',bar,mode,...config});
        const response = await fetch('/api/market?'+params, {signal:controller.signal, cache:'no-store'});
        if (!response.ok) throw new Error(response.status === 502 ? 'The market feed is unavailable. Retry or switch to Demo.' : 'Could not load market data.');
        const next = await response.json();
        if (!controller.signal.aborted) setState({key,run,data:next,error:''});
      } catch (err) {
        if (!controller.signal.aborted) setState(previous => ({...previous,key,data:previous.key===key?previous.data:null,error:err.message}));
        interval = 30000;
      } finally {
        if (!controller.signal.aborted && mode==='live') timer = setTimeout(load, interval);
      }
    }
    load();
    return () => { controller.abort(); clearTimeout(timer); };
  }, [asset,bar,mode,key,refresh,config,run]);

  return <div className="shell">
    <aside className="sidebar">
      <a className="brand" href="#overview"><span className="brand-icon">∿</span><span>TULPA<small>QUANT TERMINAL</small></span></a>
      <nav aria-label="Workspace">{nav.map(([id,label,icon]) => <a key={id} href={'#'+id}><span aria-hidden="true">{icon}</span>{label}</a>)}</nav>
      <div className="watch">
        <h2>WATCHLIST</h2>
        {assets.map(([symbol,name,icon,color]) => <button key={symbol} className={asset===symbol?'asset selected':'asset'} onClick={()=>setAsset(symbol)} aria-pressed={asset===symbol}>
          <span className="coin" style={{color}} aria-hidden="true">{icon}</span><span><b>{symbol}/USDT</b><small>{name}</small></span>
        </button>)}
      </div>
      <div className="venue"><span className="venue-icon">▦</span><span><b>OKX</b><small>Spot markets</small></span></div>
    </aside>
    <main id="overview">
      <header className="top"><span>Workspace <span className="slash">/</span> Market overview</span>
        <div className="segmented" role="group" aria-label="Data source">
          <button aria-pressed={mode==='demo'} onClick={()=>setMode('demo')}>Demo</button>
          <button aria-pressed={mode==='live'} onClick={()=>setMode('live')}>Live OKX</button>
        </div>
      </header>
      <div className="content">
        <div className="intro"><div><h1>Market overview</h1><p>Spot markets and quantitative strategy research</p></div>
          <span className={'status '+(error?'error':mode==='demo'?'demo':'live')} role="status"><i/>{error?'Feed unavailable':!data?'Loading feed':mode==='demo'?'Simulated data':'OKX public feed'}</span>
        </div>
        {error && <div className="notice" role="alert">{error}{data && ' Showing the last received snapshot.'} <button onClick={()=>setRefresh(i=>i+1)}>Retry</button></div>}
        <section className="stats" aria-label="Market snapshot">
          {[
            ['Last price',data && price(data.price),'USDT', ''],
            ['24h change',data && pct(data.change),'',data?tone(data.change):''],
            ['24h high / low',data && price(data.high),data?'Low '+price(data.low):'', ''],
            ['24h quote volume',data && compact.format(data.volume),'USDT',''],
          ].map(([label,value,unit,color])=><div className="card metric" key={label}><span>{label}</span><strong className={color}>{value ?? '—'}</strong>{unit && <small>{unit}</small>}</div>)}
        </section>
        <section className="card market" aria-labelledby="price-heading">
          <div className="panel-head"><div className="instrument"><span className="coin" style={{color:selected[3]}}>{selected[2]}</span><div><h2 id="price-heading">{asset}/USDT</h2><p>{selected[1]} · Spot</p></div></div>
          <div className="segmented" role="group" aria-label="Candle interval">{intervals.map(([value,label])=><button key={value} aria-pressed={bar===value} onClick={()=>setBar(value)}>{label}</button>)}</div></div>
          {data ? <PriceChart key={key} candles={data.candles} bar={bar}/> : <div className="placeholder">{error?'Price chart unavailable':'Loading candles…'}</div>}
          <div className="chart-foot"><span>Confirmed candles · Volume in USDT</span><span>{data?'As of '+timeFormat.format(data.asOf)+' UTC':'UTC'}</span></div>
        </section>
        <StrategyLab config={config} onApply={setConfig} result={active.run===run?data?.result:null} feedError={error}/>
      </div>
    </main>
  </div>;
}
