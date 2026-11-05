#!/usr/bin/env python3
"""Summarize balance_audit_test.go CSVs. usage: analyze.py <sim-out dir> <label> [label...]"""
import csv, glob, os, statistics as st, sys

base = sys.argv[1]
labels = sys.argv[2:]
CL = ["Warrior", "Mage", "Rogue", "Cleric"]


def load(label, kind):
    rows = []
    for p in sorted(glob.glob(os.path.join(base, label, f"{label}_c*_{kind}.csv"))):
        with open(p) as f:
            rows += list(csv.DictReader(f))
    return rows


def med(xs):
    xs = [x for x in xs if x is not None]
    return st.median(xs) if xs else None


def fmt(v, d=1):
    if v is None:
        return "—"
    return f"{v:.{d}f}"


out = []
P = out.append
marks = [10, 20, 30, 40, 50]
for L in labels:
    reach = load(L, "reach")
    P(f"\n## {L}\n")
    P("### Tempo até o andar (horas de jogo, mediana das seeds; `—` = não chegou em 48h)\n")
    P("| classe | " + " | ".join(f"f{m}" for m in marks) + " | andar máx | nível | mortes | expedições |")
    P("|---|" + "---|" * (len(marks) + 4))
    for c in CL:
        rs = [r for r in reach if r["class"] == c]
        if not rs:
            continue
        cells = []
        for m in marks:
            vals = [float(r[f"h_f{m}"]) for r in rs if r[f"h_f{m}"]]
            s = fmt(med(vals), 2) if len(vals) == len(rs) else (fmt(med(vals), 2) + f" ({len(vals)}/{len(rs)})" if vals else "—")
            cells.append(s)
        P(f"| {c} | " + " | ".join(cells) + f" | {fmt(med([int(r['max_depth']) for r in rs]),0)} | {fmt(med([int(r['level']) for r in rs]),0)} | {fmt(med([int(r['deaths']) for r in rs]),0)} | {fmt(med([int(r['expeditions']) for r in rs]),0)} |")

    floors = load(L, "floors")
    P("\n### Por faixa de andar (todas as classes e seeds somadas)\n")
    P("| andares | min jogados | mortes | mortes/h | XP/min | ouro/min | mortes nos chefes (andar 10k) |")
    P("|---|---|---|---|---|---|---|")
    for lo in range(1, 61, 5):
        hi = lo + 4
        fs = [f for f in floors if lo <= int(f["floor"]) <= hi]
        if not fs:
            continue
        sec = sum(int(f["sec"]) for f in fs)
        de = sum(int(f["deaths"]) for f in fs)
        xp = sum(int(f["xp"]) for f in fs)
        gd = sum(int(f["gold"]) for f in fs)
        boss = sum(int(f["deaths"]) for f in fs if int(f["floor"]) % 10 == 0)
        P(f"| {lo}–{hi} | {sec/60:.0f} | {de} | {de/max(sec,1)*3600:.2f} | {xp/max(sec,1)*60:.0f} | {gd/max(sec,1)*60:.1f} | {boss} |")

    # stalls: per run, gap between reaching f and f+1
    P("\n### Paredes no jogo completo (gap entre chegar em f e em f+1 > 1h; `B` = andar de chefe)\n")
    walls = {}
    for r in reach:
        pass
    # gaps from floors csv aren't enough: use hourly max_depth plateau
    hourly = load(L, "hourly")
    runs = {}
    for h in hourly:
        runs.setdefault((h["class"], h["seed"]), []).append(h)
    P("| classe | seed | andar máx parado | de h | até h | duração h | chefe? |")
    P("|---|---|---|---|---|---|---|")
    for (c, s), hs in sorted(runs.items()):
        hs.sort(key=lambda x: int(x["hour"]))
        start = 0
        for i in range(1, len(hs) + 1):
            if i == len(hs) or hs[i]["max_depth"] != hs[start]["max_depth"]:
                dur = i - start
                if dur >= 3:
                    md = int(hs[start]["max_depth"])
                    stuck = md  # the floor reached but not passed
                    P(f"| {c} | {s} | {md} | {hs[start]['hour']} | {hs[i-1]['hour']} | {dur}{'+' if i == len(hs) else ''} | {'B' if md % 10 == 0 else ''} |")
                start = i

    duels = load(L, "duels")
    P("\n### Duelo: herói real ao chegar no andar f contra monstros do próprio f (modo raw = sem poção, sem fuga; seed 1)\n")
    P("| classe | f | nv | HP | Atk | Def | vitória | TTK p10/p50/p90 (rodadas) | luta s p50 | golpes p50 | HP perdido | dano/rod dado | dano/rod levado | TTD (rod.) | XP/luta |")
    P("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|")
    for c in CL:
        ds = [d for d in duels if d["class"] == c and d["mode"] == "raw" and d["boss"] == "false" and d.get("pack", "1") == "1" and d["hero_floor"] == d["fight_floor"]]
        ds.sort(key=lambda d: int(d["hero_floor"]))
        for d in ds:
            f = int(d["hero_floor"])
            if f in (1, 5, 10, 11, 15, 20, 21, 25, 30, 31, 35, 40, 41, 45, 50, 55, 60):
                P(f"| {c} | {f} | {d['lvl']} | {d['hp']} | {d['atk']} | {d['def']} | {float(d['win']):.2f} | {float(d['ttk_p10']):.0f}/{float(d['ttk_p50']):.0f}/{float(d['ttk_p90']):.0f} | {float(d['sec_p50']):.1f} | {float(d['hits_p50']):.0f} | {float(d['hp_lost']):.2f} | {float(d['dps_dealt']):.1f} | {float(d['dps_taken']):.1f} | {float(d['ttd_rounds']):.1f} | {float(d['xp_win']):.0f} |")

    P("\n### Grupos: herói de entrada contra 1, 2 e 3 monstros do andar (vitória raw / vitória kit; HP perdido raw no grupo de 3)\n")
    P("| classe | f | 1 | 2 | 3 | HP perdido (3) |")
    P("|---|---|---|---|---|---|")
    for c in CL:
        for fl in (5, 10, 15, 20, 25, 30, 35, 40, 45):
            g = lambda k, m: next((d for d in duels if d["class"] == c and d["boss"] == "false" and d["mode"] == m and d.get("pack") == str(k) and int(d["hero_floor"]) == fl and d["hero_floor"] == d["fight_floor"]), None)
            if g(1, "raw") is None:
                continue
            cells = [f"{float(g(k,'raw')['win']):.2f} / {float(g(k,'kit')['win']):.2f}" for k in (1, 2, 3)]
            P(f"| {c} | {fl} | " + " | ".join(cells) + f" | {float(g(3,'raw')['hp_lost']):.2f} |")

    P("\n### Chefes: herói ao chegar no andar do chefe (entrada) e ao chegar no andar seguinte (o que venceu); kit = 3 poções + fuga\n")
    P("| classe | andar | nv entrada→venceu | Atk | Def | vitória raw entrada | vitória kit entrada | vitória raw venceu | TTK p50 venceu | horas parado no chefe (mediana) |")
    P("|---|---|---|---|---|---|---|---|---|---|")
    ra = load(L, "reachall")
    for c in CL:
        bs = [d for d in duels if d["class"] == c and d["boss"] == "true"]
        for f in sorted({int(d["fight_floor"]) for d in bs}):
            g = lambda hf, mode: next((d for d in bs if int(d["fight_floor"]) == f and int(d["hero_floor"]) == hf and d["mode"] == mode), None)
            er, ek, br = g(f, "raw"), g(f, "kit"), g(f + 1, "raw")
            gaps = []
            for seed in {r["seed"] for r in ra if r["class"] == c}:
                rr = {int(r["floor"]): int(r["sec"]) for r in ra if r["class"] == c and r["seed"] == seed}
                if f in rr and f + 1 in rr:
                    gaps.append((rr[f + 1] - rr[f]) / 3600)
            P(f"| {c} | {f} | {er['lvl']}→{br['lvl'] if br else '—'} | {er['atk']}→{br['atk'] if br else '—'} | {er['def']}→{br['def'] if br else '—'} | {float(er['win']):.2f} | {float(ek['win']):.2f} | {float(br['win']) if br else float('nan'):.2f} | {float(br['ttk_p50']) if br else float('nan'):.0f} | {fmt(med(gaps),2)} ({len(gaps)} seeds) |")

    P("\n### Gradiente: herói congelado ao chegar em f, lutando em f+k (raw, vitória / TTK p50)\n")
    P("| classe | f | " + " | ".join(f"+{k}" for k in range(0, 10)) + " |")
    P("|---|---|" + "---|" * 10)
    for c in CL:
        for f in (1, 11, 21, 31, 41):
            ds = {int(d["fight_floor"]) - f: d for d in duels if d["class"] == c and d["mode"] == "raw" and d["boss"] == "false" and d.get("pack", "1") == "1" and int(d["hero_floor"]) == f}
            if len(ds) < 2:
                continue
            P(f"| {c} | {f} | " + " | ".join(f"{float(ds[k]['win']):.2f}/{float(ds[k]['ttk_p50']):.0f}" if k in ds else "" for k in range(10)) + " |")

    # automatic wall detection on the entry-hero duel curve
    P("\n### Detecção automática (duelo herói-de-entrada vs andar; queda de vitória ≥ 15 p.p. ou TTK p50 ×1,5 e +2 rodadas ou TTD ÷1,5 entre andares seguidos)\n")
    P("| classe | andar | sinal | antes → depois | chefe? |")
    P("|---|---|---|---|---|")
    for c in CL:
        ds = {int(d["hero_floor"]): d for d in duels if d["class"] == c and d["mode"] == "raw" and d["boss"] == "false" and d.get("pack", "1") == "1" and d["hero_floor"] == d["fight_floor"]}
        for f in sorted(ds):
            if f - 1 not in ds:
                continue
            a, b = ds[f - 1], ds[f]
            sig = []
            if float(a["win"]) - float(b["win"]) >= 0.15:
                sig.append(("vitória", f"{float(a['win']):.2f} → {float(b['win']):.2f}"))
            if float(a["ttk_p50"]) > 0 and float(b["ttk_p50"]) / float(a["ttk_p50"]) >= 1.5 and float(b["ttk_p50"]) - float(a["ttk_p50"]) >= 2:
                sig.append(("TTK", f"{float(a['ttk_p50']):.0f} → {float(b['ttk_p50']):.0f}"))
            ta, tb = float(a["ttd_rounds"]), float(b["ttd_rounds"])
            if ta != float("inf") and tb > 0 and ta / tb >= 1.5 and tb < 15:
                sig.append(("TTD", f"{ta:.1f} → {tb:.1f}"))
            for s, v in sig:
                P(f"| {c} | {f} | {s} | {v} | {'B (+1 após chefe)' if f % 10 == 1 else ''} |")

open(os.path.join(base, "summary_" + "_".join(labels) + ".md"), "w").write("\n".join(out) + "\n")
print("\n".join(out))
