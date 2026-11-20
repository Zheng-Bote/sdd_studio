---
type: Note
title: Kombination von SpecDD, Spec Kit und BMAD
description: BMAD method und SpecKit und SpecDD kombinieren
status: ""
notebook: ""
tags:
  - BMAD
  - SpecKit
  - SDD
  - SpecDD
uses: []
depends_on: []
contains: []
references:
adr:
  - https://github.com/github/spec-kit
  - https://github.com/bmad-code-org/BMAD-METHOD
relation: []
api: ""
creation date: 2026-11-18 02:03
modification date: Mittwoch 18. November 2026 02:03:39
---
# BMAD method und SpecKit kombinieren

Um die **BMAD-Method** und [GitHub Spec Kit](https://github.github.com/spec-kit/) effektiv zu kombinieren, nutzt du **BMAD für die vorgelagerte Strategie, Konzeption und Rollen-Simulation**

(Makro-Ebene) und **Spec Kit für die strukturierte, verifizierbare Feature-Ausführung** (Mikro-Ebene). [[1](https://github.com/Koroqe/claude-code-sdlc/blob/main/docs/guides/bmad-method-vs-spec-kit.md), [2](https://medium.com/@visrow/a-tale-of-two-frameworks-bmad-method-vs-github-spec-kit-c021ab0ad037), [3](https://reenbit.com/bmad-vs-spec-kit-vs-openspec-choosing-your-spec-driven-ai-framework/)]

Während BMAD ein komplettes agiles Team (Analyst, PM, Architect) simuliert, um tiefe Planungsartefakte zu erzeugen, sorgt Spec Kit mit einer projektweiten Verfassung () und klaren Befehlen (`/specify`, `/plan`, `/tasks`, `/implement`) dafür, dass der Coding-Agent bei der Umsetzung nicht vom Kurs abkommt. [[1](https://speckit.org/), [2](https://www.youtube.com/watch?v=sGYvGUkerA0), [3](https://www.exaltech.it/en/blog/bmad-method-ai-development), [4](https://reenbit.com/bmad-vs-spec-kit-vs-openspec-choosing-your-spec-driven-ai-framework/)]


Rollenverteilung im hybriden Workflow

|Phase|Zuständiges Framework|Kernaufgabe & Output|
|---|---|---|
|**1. Discovery & Produktvision**|**BMAD-Method**|Analyst- und PM-Agenten erarbeiten Produktziele, User-Flows und ein grobes PRD (Product Requirements Document).|
|**2. Systemarchitektur & Regeln**|**BMAD + Spec Kit**|Der BMAD-Architect definiert Tech-Stack und Datenmodelle. Diese Leitplanken werden in die von Spec Kit überführt.|
|**3. Feature-Spezifikation**|**GitHub Spec Kit**|Einzelne Epics aus BMAD werden per `/specify` und `/clarify` in eindeutige, testbare Feature-Specs übersetzt.|
|**4. Technische Planung & Tasks**|**GitHub Spec Kit**|`/plan` und `/tasks` zerlegen die Spec in kleinteilige, maschinenlesbare Arbeitspakete ([tasks.md](http://tasks.md/)).|
|**5. Implementierung & Review**|**Spec Kit + BMAD QA**|`/implement` schreibt den Code; optional übernimmt der BMAD-QA-Agent abschließend das Code-Review und Testing.|

## Schritt-für-Schritt-Anleitung für die Kombination

1. Projekt-Setup im Repository

Initialisiere beide Tools im selben Projektverzeichnis, sodass sie auf dieselbe Codebasis zugreifen können: [[1](https://blog.lpains.net/posts/2025-12-07-deep-dive-into-speckit/), [2](https://medium.com/@visrow/what-is-bmad-method-a-simple-guide-to-the-future-of-ai-driven-development-412274f91419)]

- **BMAD installieren:** Richte die Rollen-Personas über das [BMAD-METHOD Repository](https://github.com/bmad-code-org/BMAD-METHOD/blob/main/README.md) (`npx bmad-method install`) im Ordner `.bmad-core/` ein.
- **Spec Kit initialisieren:** Installiere das CLI über das [GitHub Spec Kit Repository](https://github.com/github/spec-kit) (`specify init`).
- _Alternativ:_ Auf GitHub existiert mit bereits ein Community-Fork, der beide Ansätze in einer Struktur vorintegriert. [[1](https://github.com/github/spec-kit), [2](https://github.com/bmad-code-org/bmad-method), [3](https://github.com/oimiragieo/BMAD-SPEC-KIT), [4](https://medium.com/@visrow/what-is-bmad-method-a-simple-guide-to-the-future-of-ai-driven-development-412274f91419), [5](https://blog.lpains.net/posts/2025-12-07-deep-dive-into-speckit/)]

2. Makro-Planung mit BMAD (Einmalig zum Projektstart)

Nutze die spezialisierten Agenten von BMAD, um komplexe Systementscheidungen abzusichern, bevor Code entsteht: [[1](https://dev.to/bspann/bmad-method-ai-driven-agile-development-that-actually-works-part-1-core-framework-36n1), [2](https://www.youtube.com/watch?v=4ytALcI-yt0), [3](https://www.exaltech.it/en/blog/bmad-method-ai-development)]

- **Ideation & PRD:** Lass den **Business Analyst** und **Product Manager** die Anforderungen schärfen und Epics schneiden.
- **Architektur:** Lass den **Architect-Agenten** die Systemarchitektur, API-Schnittstellen und Datenbankschemas entwerfen. [[1](https://www.codemag.com/Article/268061/Professional-Grade-AI-Assisted-Coding-Context-Is-Everything-with-BMAD-and-Spec-Kit), [2](https://www.exaltech.it/en/blog/bmad-method-ai-development), [3](https://dev.to/bspann/bmad-method-ai-driven-agile-development-that-actually-works-part-1-core-framework-36n1)]

3. Architektur in Spec Kit verankern (`constitution.md`)

Das größte Risiko bei reinem BMAD ist der „Context Drift“ über viele Agenten-Übergaben hinweg. Löse das, indem du die wichtigsten BMAD-Architekturentscheidungen (Tech-Stack, Coding-Standards, Sicherheitsvorgaben) in die von Spec Kit überträgst. Jede spätere Spec erbt diese Regeln automatisch. [[1](https://www.youtube.com/watch?v=xnYRkHNAKr4&t=1), [2](https://reenbit.com/bmad-vs-spec-kit-vs-openspec-choosing-your-spec-driven-ai-framework/), [3](https://blog.lpains.net/posts/2025-12-07-deep-dive-into-speckit/)]

4. Feature-Umsetzung mit Spec Kit (Iterativ pro Epic/Story)

Anstatt jede kleine Story durch den zeitintensiven BMAD-Entwicklungszyklus zu schleusen, übergibst du die einzelnen BMAD-Epics an die schlankere Spec-Kit-Pipeline: [[1](https://www.viadee.de/blog/github-spec-kit-die-schlankere-alternative-zu-bmad-und-wann-es-die-bessere-wahl-ist/), [2](https://www.youtube.com/watch?v=sGYvGUkerA0), [3](https://reenbit.com/bmad-vs-spec-kit-vs-openspec-choosing-your-spec-driven-ai-framework/)]

- **`/specify`:** Füttere den Befehl mit einem Epic oder einer User Story aus deinem BMAD-PRD.
- **`/clarify`:** Kläre offene Randfälle direkt mit dem Coding-Agenten (z. B. Claude Code, Cursor oder GitHub Copilot).
- **`/plan` & `/tasks`:** Generiere den konkreten Implementierungsplan und die Checkliste.
- **`/implement`:** Lasse den Code auf Basis der verifizierten Spezifikation generieren. [[1](https://wavect.io/de/blog/github-spec-kit-production-guide/), [2](https://dev.to/bspann/bmad-method-ai-driven-agile-development-that-actually-works-part-1-core-framework-36n1), [3](https://github.com/Koroqe/claude-code-sdlc/blob/main/docs/guides/bmad-method-vs-spec-kit.md), [4](https://medium.com/@visrow/a-tale-of-two-frameworks-bmad-method-vs-github-spec-kit-c021ab0ad037), [5](https://speckit.org/), [6](https://www.codemag.com/Article/268061/Professional-Grade-AI-Assisted-Coding-Context-Is-Everything-with-BMAD-and-Spec-Kit)]

5. Qualitätssicherung mit dem BMAD-QA-Agenten

Nach dem Durchlauf von `/implement` kannst du den **BMAD-QA-Agenten** auf den erzeugten Code und die Spec ansetzen, um Edge-Cases, Nicht-Funktionale Anforderungen und Testabdeckung zu prüfen. [[1](https://www.youtube.com/watch?v=sGYvGUkerA0), [2](https://www.exaltech.it/en/blog/bmad-method-ai-development)]

---

Wann sich diese Kombination lohnt

- **Ideal für:** Komplexe Greenfield-Projekte, Multi-Service-Architekturen oder Teams, die am Anfang tiefe fachliche und architektonische Klarheit brauchen (BMAD), im Alltag aber schnelle, token-effiziente Feature-Iterationen bevorzugen (Spec Kit).
- **Overkill für:** Kleine Bugfixes oder isolierte Änderungen in bestehenden Codebasen – hier reicht Spec Kit allein völlig aus. [[1](https://gist.github.com/lukasjsk/7b8d950091aef74b31dcd2216c4acb6d), [2](https://github.com/github/spec-kit), [3](https://blog.lpains.net/posts/2025-12-07-deep-dive-into-speckit/), [4](https://reenbit.com/bmad-vs-spec-kit-vs-openspec-choosing-your-spec-driven-ai-framework/), [5](https://www.youtube.com/watch?v=xnYRkHNAKr4&t=1)]



### **Kombination von SpecDD, Spec Kit und BMAD**

- **BMAD** wird für **Makro-Planung** und **Systemarchitektur** verwendet, während **Spec Kit** für die **Feature-Implementierung** und **schnelle Iterationen** genutzt wird [Doc: /Volumes/1TB_NVME/Documents/obsidian-snippets/BMAD method und SpecKit kombinieren.md#BMAD method und SpecKit kombinieren].
- **SpecDD** definiert die **systemweiten Regeln** (z. B. Sicherheitsvorgaben), die **Spec Kit** automatisch erbt [Doc: /Volumes/1TB_NVME/Documents/obsidian-snippets/BMAD method und SpecKit kombinieren.md#Schritt-für-Schritt-Anleitung für die Kombination].
- **Integration Matrix:**
    - **SpecDD** ist verantwortlich für **System-Intent**, **Architecture**, **Governance / Security**.
    - **Spec Kit** ist verantwortlich für **Feature-Intent**, **Feature Planning**, **Feature Implementation**.
    - Beide Tools sind für **Drift Control** verantwortlich [Doc: /Volumes/1TB_NVME/Documents/obsidian-snippets/SpecDD_SpecKit_Integration_EN.md#5. Integration Matrix].

### 5. **Vorteile der Kombination**

- **BMAD** sorgt für **tiefe fachliche und architektonische Klarheit** am Projektstart.
- **Spec Kit** ermöglicht **schnelle, token-effiziente Feature-Iterationen** im Alltag.
- **SpecDD** sorgt für **stabile, langfristige Systemarchitektur** und **Sicherheitsvorgaben**.