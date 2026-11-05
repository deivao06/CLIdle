import csv, glob, statistics as st, sys
B = sys.argv[1] if len(sys.argv) > 1 else "/tmp/clidle-audit/sim-out"
M = ["A-exp", "B35", "B40", "B45"]
def rows(m, k):
    out = []
    for p in sorted(glob.glob(f"{B}/{m}/{m}_c*_{k}.csv")):
        out += list(csv.DictReader(open(p)))
    return out
print("## Duelo 1x1, perfil 'adequate' (mediana das 4 classes): golpes p/ matar | rodadas p/ morrer | % HP perdido | vitória")
fl = [10, 20, 30, 40, 50, 60]
print("| modelo | " + " | ".join(f"andar {f}" for f in fl) + " |")
for m in M:
    s = [r for r in rows(m, "synth") if r["profile"] == "adequate"]
    cells = []
    for f in fl:
        x = [r for r in s if int(r["floor"]) == f]
        g = lambda k: st.median(float(r[k]) for r in x)
        cells.append(f"{g('ttk_p50'):.0f} / {g('ttd_rounds'):.0f} / {100*g('hp_lost'):.0f}% / {g('win'):.2f}")
    print(f"| {m} | " + " | ".join(cells) + " |")
for prof in ["under", "over"]:
    print(f"\n## Vitória 1x1, perfil '{prof}' (mediana classes)")
    print("| modelo | " + " | ".join(f"{f}" for f in fl) + " |")
    for m in M:
        s = [r for r in rows(m, "synth") if r["profile"] == prof]
        print(f"| {m} | " + " | ".join(f"{st.median(float(r['win']) for r in s if int(r['floor'])==f):.2f}" for f in fl) + " |")
print("\n## Jogo completo: mortes por 100 monstros mortos, por faixa (todas classes/seeds) e % de mortes em chefe")
bands = [(1,10),(11,20),(21,30),(31,40),(41,50),(51,60),(61,80)]
print("| modelo | " + " | ".join(f"{a}–{b}" for a,b in bands) + " | % mortes em chefe | andar máx (mediana) | spread classes andar máx |")
for m in M:
    fr = rows(m, "floors")
    cells = []
    for a, b in bands:
        x = [r for r in fr if a <= int(r["floor"]) <= b]
        k = sum(int(r["kills"]) for r in x); d = sum(int(r["deaths"]) for r in x)
        cells.append("—" if k == 0 else f"{100*d/k:.1f}")
    td = sum(int(r["deaths"]) for r in fr); bd = sum(int(r["deaths"]) for r in fr if int(r["floor"]) % 10 == 0)
    rc = rows(m, "reach")
    mx = [int(r["max_depth"]) for r in rc] if rc and "max_depth" in rc[0] else []
    bycls = {}
    for r in rc:
        bycls.setdefault(r["class"], []).append(int(r.get("max_depth", 0)))
    meds = [st.median(v) for v in bycls.values()]
    print(f"| {m} | " + " | ".join(cells) + f" | {100*bd/max(td,1):.0f}% | {st.median(mx) if mx else '?'} | {min(meds) if meds else '?'}–{max(meds) if meds else '?'} |")
