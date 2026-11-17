package game

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// simFormula is one balance formula as tools/simview shows it above a chart: the expression, what each number
// means and a few values it gives.
type simFormula struct {
	Name     string      `json:"name"`
	Expr     string      `json:"expr"`
	Terms    [][2]string `json:"terms"`
	Examples [][2]string `json:"examples"`
}

// ptNum writes a number the Brazilian way, as tools/simview reads.
func ptNum(v float64) string {
	if v == float64(int64(v)) {
		s := fmt.Sprint(int64(v))
		for i := len(s) - 3; i > 0 && s[i-1] != '-'; i -= 3 {
			s = s[:i] + "." + s[i:]
		}
		return s
	}
	return strings.ReplaceAll(strings.TrimRight(fmt.Sprintf("%.2f", v), "0"), ".", ",")
}

// simFormulas are the balance formulas in force, keyed by the simview metrics they shape.
func simFormulas() map[string][]simFormula {
	xpEx := [][2]string{}
	for _, l := range []int{5, 10, 20, 30, 40} {
		xpEx = append(xpEx, [2]string{fmt.Sprintf("nível %d", l), ptNum(float64(XPToNext(l))) + " XP"})
	}
	trainEx := [][2]string{}
	for _, n := range []int{5, 10, 20, 30} {
		trainEx = append(trainEx, [2]string{fmt.Sprintf("%dº ponto", n+1), ptNum(float64(TrainCost(n))) + " ouro"})
	}
	goldEx := [][2]string{}
	for _, d := range []int{10, 30, 60, 100} {
		goldEx = append(goldEx, [2]string{fmt.Sprintf("andar %d", d), ptNum(float64(int(GoldPerPile(d)))) + " ouro"})
	}
	monEx := [][2]string{}
	for _, n := range []int{10, 30, 60} {
		monEx = append(monEx, [2]string{fmt.Sprintf("%d andares abaixo", n), "×" + ptNum(MonsterScale(n, 0))})
	}
	bossEx := [][2]string{}
	for _, d := range []int{10, 30, 60, 100} {
		k := float64(d/BossEvery - 1)
		bossEx = append(bossEx, [2]string{fmt.Sprintf("andar %d", d), fmt.Sprintf("HP ×%s · Ataque ×%s",
			ptNum(BossHPMult+BossHPPerBoss*k), ptNum(BossAtkMult+BossAtkPerBoss*k))})
	}
	monster := simFormula{
		Name:     "Força dos monstros",
		Expr:     fmt.Sprintf("multiplicador = 1 + %s × andares abaixo do primeiro andar do monstro", ptNum(MonsterGrowthLin)),
		Terms:    [][2]string{{"1", "o monstro no primeiro andar em que aparece"}, {ptNum(MonsterGrowthLin), "quanto HP, ataque, defesa e XP crescem a cada andar (linear)"}},
		Examples: monEx,
	}
	boss := simFormula{
		Name: "Chefes",
		Expr: fmt.Sprintf("HP × (%s + %s × chefes antes) · Ataque × (%s + %s × chefes antes) · Defesa × %s",
			ptNum(BossHPMult), ptNum(BossHPPerBoss), ptNum(BossAtkMult), ptNum(BossAtkPerBoss), ptNum(BossDefMult)),
		Terms: [][2]string{
			{ptNum(BossHPMult), "HP do chefe do andar 10 em relação ao monstro mais forte do andar"},
			{ptNum(BossHPPerBoss), "HP extra a cada chefe seguinte (soma com o crescimento dos monstros, por isso cresce como quadrado)"},
			{ptNum(BossAtkMult), "ataque do primeiro chefe em relação ao monstro"},
			{ptNum(BossAtkPerBoss), "ataque extra a cada chefe seguinte"},
			{ptNum(BossDefMult), "defesa do chefe em relação ao monstro (fixa)"},
		},
		Examples: bossEx,
	}
	gold := simFormula{
		Name:     "Ouro por pilha",
		Expr:     fmt.Sprintf("ouro = %s × %s^(andar − 1)", ptNum(GoldPerPile(1)), ptNum(GoldPerPile(2)/GoldPerPile(1))),
		Terms:    [][2]string{{ptNum(GoldPerPile(1)), "ouro de uma pilha no andar 1"}, {ptNum(GoldPerPile(2) / GoldPerPile(1)), "cada andar dá 8% a mais que o anterior (exponencial)"}},
		Examples: goldEx,
	}
	return map[string][]simFormula{
		"level": {{
			Name: "XP para o próximo nível",
			Expr: fmt.Sprintf("XP = %d + %d × (nível − 1)²", XPBase, XPQuad),
			Terms: [][2]string{{fmt.Sprint(XPBase), "XP para subir do nível 1 para o 2 (a base)"},
				{fmt.Sprint(XPQuad), "quanto o custo cresce com o quadrado do nível (quadrático)"}},
			Examples: xpEx,
		}},
		"train": {{
			Name: "Preço do treino",
			Expr: fmt.Sprintf("preço = %d × %s^(pontos já treinados no atributo)", TrainBase, ptNum(float64(TrainCost(1))/float64(TrainCost(0)))),
			Terms: [][2]string{{fmt.Sprint(TrainBase), "preço do primeiro ponto de um atributo"},
				{ptNum(float64(TrainCost(1)) / float64(TrainCost(0))), "cada ponto custa 50% a mais que o anterior (exponencial)"}},
			Examples: trainEx,
		}},
		"gold":   {gold},
		"spent":  {gold},
		"floor":  {monster, boss},
		"deaths": {monster, boss},
	}
}

// writeSimFormulas saves the formulas in force next to a simulation's files, once per batch.
func writeSimFormulas(batch string) {
	dir := filepath.Join("..", "..", "tools", "simview", "data", batch)
	path := filepath.Join(dir, "_formulas.json")
	if _, err := os.Stat(path); err == nil {
		return
	}
	os.MkdirAll(dir, 0o755)
	data, _ := json.Marshal(simFormulas())
	os.WriteFile(path, data, 0o644)
}

// TestSimFormulas writes the formulas file for SIMVIEW_BATCH, for a simulation started before the file existed.
func TestSimFormulas(t *testing.T) {
	if b := os.Getenv("SIMVIEW_BATCH"); b != "" && os.Getenv("SIM_FORMULAS") != "" {
		writeSimFormulas(b)
	} else {
		t.Skip()
	}
}
