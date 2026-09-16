import sys; sys.path.insert(0,'/home/poezd/work/emergency/new_core/trainer')
from pathlib import Path
from caller.scenario import Scenario
from caller.engine import Engine
from caller.contracts import Style
import statistics
out = open('/home/poezd/work/emergency/new_core/trainer/work/sweep.txt','w',encoding='utf-8')
rows=[]
for scp in sorted(Path("/home/poezd/work/emergency/new_core/trainer/scenarios").glob("*.json")):
    sid = scp.stem
    bp = Path(f"/home/poezd/work/emergency/new_core/trainer/blind/blind_{sid}.txt")
    if sid=="bilet04_call01" or not bp.exists(): continue
    sc = Scenario.load(scp)
    phrases=[l.strip().lstrip("-•*0123456789. ").strip() for l in bp.read_text(encoding="utf-8").splitlines()]
    phrases=[p for p in phrases if p and not p.startswith("#")]
    fact=ack=dk=mh=0; per={k:0 for k in sc.facts}; unc=[]
    for p in phrases:
        e=Engine(sc,"lexical"); t=e.handle(p); st=t.decision.style
        if t.decision.reveal:
            fact+=1
            for k in t.decision.reveal: per[k]+=1
        elif st is Style.ACK: ack+=1
        elif st is Style.MISHEAR: mh+=1
        else:
            dk+=1; unc.append(p)
    n=len(phrases)
    critGot=[k for k in sc.critical if per[k]>0]
    zero=[k for k,v in per.items() if v==0]
    rows.append((100*fact//n if n else 0, sid, n, fact, ack, dk, mh, len(critGot), len(sc.critical), zero, unc))
    out.write(f"{100*fact//n if n else 0:>3}% {sid} crit={len(critGot)}/{len(sc.critical)} zero={','.join(zero) if zero else '-'}\n")
    out.flush()
rows.sort()
out.write(f"медиана: {statistics.median(r[0] for r in rows)} среднее: {round(sum(r[0] for r in rows)/len(rows),1)}\n")
out.close()
print("done", len(rows))
