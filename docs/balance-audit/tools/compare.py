#!/usr/bin/env python3
"""Headline comparison across models. usage: compare.py <sim-out dir> label..."""
import csv, glob, os, statistics as st, sys

base, labels = sys.argv[1], sys.argv[2:]
CL = ["Warrior", "Mage", "Rogue", "Cleric"]


def load(label, kind):
    rows = []
    for p in sorted(glob.glob(os.path.join(base, label, f"{label}_c*_{kind}.csv"))):
        with open(p) as f:
            rows += list(csv.DictReader(f))
    return rows


def med(xs):
    return st.median(xs) if xs else None


def f(v, d=1):
    return "—" if v is None else f"{v:.{d}f}"


out = []
P = out.append
P("| modelo | h f10 | h f20 | h f30 | h f40 | h f50 | andar máx 48h (min–máx) | mortes/h | % mortes em chefe | spread classes f30 (máx/mín) | vitória entrada f21–40 | TTK p50 entrada f1–40 (mín–máx) | andares até vitória<50% (herói congelado em 21) | vitória grupo de 3 f21–40 raw / kit | chefes 20/30/40: vitória do herói que venceu (mediana classes) |")
P("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|")
for L in labels:
    ra = load(L, "reachall")
    reach = {}
    for r in ra:
        reach.setdefault((r["class"], r["seed"]), {})[int(r["floor"])] = int(r["sec"]) / 3600
    runs = list(reach.values())
    cells = []
    for m in (10, 20, 30, 40, 50):
        vals = [r[m] for r in runs if m in r]
        cells.append(f(med(vals), 2) + ("" if len(vals) == len(runs) else f" ({len(vals)}/{len(runs)})") if vals else "—")
    maxd = [max(r) for r in runs]
    floors = load(L, "floors")
    sec = sum(int(x["sec"]) for x in floors)
    deaths = sum(int(x["deaths"]) for x in floors)
    bossd = sum(int(x["deaths"]) for x in floors if int(x["floor"]) % 10 == 0)
    per30 = []
    for c in CL:
        v = [reach[k][30] for k in reach if k[0] == c and 30 in reach[k]]
        if v:
            per30.append(med(v))
    spread = max(per30) / min(per30) if len(per30) == 4 else None
    duels = load(L, "duels")
    entry = [d for d in duels if d["mode"] == "raw" and d["boss"] == "false" and d.get("pack", "1") == "1" and d["hero_floor"] == d["fight_floor"]]
    win2140 = [float(d["win"]) for d in entry if 21 <= int(d["hero_floor"]) <= 40]
    ttk = [float(d["ttk_p50"]) for d in entry if int(d["hero_floor"]) <= 40 and d["ttk_p50"] not in ("NaN", "nan")]
    grad = []
    for c in CL:
        ds = {int(d["fight_floor"]) - 21: float(d["win"]) for d in duels if d["class"] == c and d["mode"] == "raw" and d["boss"] == "false" and d.get("pack", "1") == "1" and int(d["hero_floor"]) == 21}
        k = next((k for k in sorted(ds) if ds[k] < 0.5), None)
        grad.append(str(k) if k is not None else ">9" if ds else "—")
    p3 = {m: [float(d["win"]) for d in duels if d["mode"] == m and d["boss"] == "false" and d.get("pack") == "3" and d["hero_floor"] == d["fight_floor"] and 21 <= int(d["hero_floor"]) <= 40] for m in ("raw", "kit")}
    bw = []
    for bf in (20, 30, 40):
        v = [float(d["win"]) for d in duels if d["boss"] == "true" and d["mode"] == "raw" and int(d["fight_floor"]) == bf and int(d["hero_floor"]) == bf + 1]
        bw.append(f(med(v), 2) if v else "—")
    extra = f" | {f(med(p3['raw']),2)} / {f(med(p3['kit']),2)} | {'/'.join(bw)} |"
    P(f"| {L} | " + " | ".join(cells) + f" | {f(med(maxd),0)} ({min(maxd)}–{max(maxd)}) | {deaths/max(sec,1)*3600:.1f} | {100*bossd/max(deaths,1):.0f}% | {f(spread,2)} | {f(med(win2140),2)} | {f(min(ttk) if ttk else None,0)}–{f(max(ttk) if ttk else None,0)} | {'/'.join(grad)}" + extra)
print("\n".join(out))
open(os.path.join(base, "compare.md"), "w").write("\n".join(out) + "\n")
