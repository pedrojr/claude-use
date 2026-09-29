# claude-use

[Read in English](README.md)

Overlay compacto e semi-transparente para Windows que mostra os limites de uso do Claude
(o mesmo que a tela **Configurações → Uso** / `/usage` do Claude Code). Feito em Go com [Fyne](https://fyne.io).

- Fica encostado no lado direito do monitor principal, centralizado na vertical, sempre por cima.
- Arraste com o mouse para mover para qualquer lugar; a partir daí ele sempre abre nessa posição.
  **Restaurar posição** no menu da bandeja volta a encostá-lo no lado direito.
- Sem borda, cantos arredondados, ~85% opaco, textos em branco, sem botão na barra de tarefas (só o ícone da bandeja) e não rouba o foco.
- Apenas uma instância: abrir o executável de novo com o overlay já aberto não faz nada.
- Atualiza a cada 5 minutos, logo após um limite reiniciar e 1 minuto depois de uma tentativa sem
  conexão (por exemplo, quando o PC volta da suspensão); o botão ⟳ discreto no rodapé força a atualização.
- A contagem "Reinicia em…" é recalculada localmente a cada 30 s.
- Barras mudam de cor: azul, laranja (≥ 75%) e vermelho (≥ 90%).
- Interface em inglês (padrão) ou português do Brasil.

## Bandeja do sistema

Menu do ícone na bandeja:

- **Atualizar agora**
- **Ignorar cliques (atravessar)** – os cliques passam através do overlay (a transparência não muda).
  Use a bandeja para desativar e voltar a clicar no botão de refresh.
- **Ocultar / Mostrar**
- **Restaurar posição** – esquece a posição para onde o overlay foi arrastado.
- **Iniciar com o Windows** – liga/desliga o início automático para o usuário atual (grava o caminho do `.exe`
  atual em `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`). Se mover o executável, desative e ative de novo.
- **Idioma** – English ou Português (Brasil). Aplicado na hora e lembrado.
- **Sair**

## Configurações

As configurações ficam por usuário em `%APPDATA%\claude-use\config.json`, nunca ao lado do executável,
então funcionam com o app instalado em Arquivos de Programas (somente leitura para usuários comuns):

```json
{
  "language": "pt-BR",
  "position": { "x": 1500, "y": 300 }
}
```

- `language`: `en` (padrão) ou `pt-BR`.
- `position`: canto superior esquerdo do overlay, em pixels da tela. Ausente = encostado no lado direito.
  Se a disposição dos monitores mudar, o overlay é mantido dentro da tela mais próxima.

Apague o arquivo para voltar ao padrão. A única outra coisa gravada é o valor de **Iniciar com o Windows**
em `HKCU\...\Run`. O Fyne (toolkit da interface) também cria a pasta vazia `%APPDATA%\fyne\com.github.claude-use`;
nada é guardado nela.

## Instalação

Baixe o instalador `claude-use-setup-<versão>.exe` na página de
[Releases](https://github.com/pedrojr/claude-use/releases/latest) e execute. Ele:

- pergunta como instalar:
  - **Instalar para todos os usuários** (padrão) – em `C:\Program Files\claude-use` (Arquivos de Programas),
    exige administrador (UAC);
  - **Instalar apenas para mim** – em `%LOCALAPPDATA%\Programs\claude-use`, não precisa de administrador
    (use esta opção em máquinas onde você não é administrador);
- permite alterar a pasta de destino nos dois modos. Sem administrador, verifica se a pasta escolhida
  pode ser gravada antes de instalar;
- cria atalho no Menu Iniciar (e, se marcado, na Área de Trabalho);
- oferece a opção **Iniciar com o Windows** (a mesma do menu da bandeja), registrada para o usuário que abriu
  o instalador, mesmo quando a elevação (UAC) foi feita com outra conta;
- pede para fechar o overlay se ele estiver aberto (útil ao atualizar);
- ao instalar para todos os usuários, remove uma instalação por usuário já existente (versões anteriores só instalavam assim);
- ao desinstalar, remove também o início automático se ele apontar para aquela cópia. Suas configurações em
  `%APPDATA%\claude-use` são mantidas.

Instalação silenciosa: `claude-use-setup-<versão>.exe /VERYSILENT /ALLUSERS` (ou `/CURRENTUSER`); `/DIR="C:\pasta"`
muda a pasta. Para gerar log: `/LOG="%TEMP%\claude-use-setup.log"`.

Na mesma release há o `claude-use.exe` avulso, que roda sem instalar.

## Requisitos (para compilar)

- Go 1.26+ (veja `go.mod`)
- GCC para CGO (ex.: [MinGW-w64 / WinLibs](https://winlibs.com)) – exigido pelo Fyne
- Claude Code logado com conta Claude (Pro/Max/Team/Enterprise)

## Build

```bash
windres -O coff -o rsrc_windows_amd64.syso claude-use.rc   # ícone do executável (opcional)
go build -ldflags "-H=windowsgui -s -w" -o claude-use.exe .
```

O `windres` vem com o MinGW. O ícone é [assets/claude-use.ico](assets/claude-use.ico), gerado a partir do
ícone da bandeja com `go run assets/mkicon.go`. `-H=windowsgui` evita abrir console. Para testar sem interface:

```bash
go build -o claude-use-cli.exe . && ./claude-use-cli.exe -print
```

Parâmetros de linha de comando:

- `-print` – imprime o uso no terminal e sai.
- `-autostart=on|off` – liga/desliga **Iniciar com o Windows** para o usuário atual e sai (usado pelo instalador).

### Instalador

O instalador é feito com [Inno Setup](https://jrsoftware.org/isinfo.php) (script em
[installer/claude-use.iss](installer/claude-use.iss)). Localmente:

```bash
windres -O coff -o rsrc_windows_amd64.syso claude-use.rc
go build -trimpath -ldflags "-H=windowsgui -s -w" -o dist/claude-use.exe .
iscc /DAppVersion=1.0.0 installer/claude-use.iss   # gera dist/claude-use-setup-1.0.0.exe
```

## Publicar uma versão

O workflow [.github/workflows/release.yml](.github/workflows/release.yml) compila o app e o instalador
no GitHub Actions. Ao enviar uma tag `v*`, ele cria a release com os dois arquivos:

```bash
git tag v1.0.0
git push origin v1.0.0
```

Também dá para rodar o workflow manualmente (aba **Actions → Release → Run workflow**). Nesse caso
ele só compila e deixa os arquivos como artifact da execução, sem criar release.

## Como os dados são obtidos

O app lê o token OAuth que o Claude Code guarda em `%USERPROFILE%\.claude\.credentials.json`
(ou `%CLAUDE_CONFIG_DIR%\.credentials.json`) e chama `GET https://api.anthropic.com/api/oauth/usage`,
o mesmo endpoint da tela de uso. O arquivo é relido a cada atualização; o token nunca é gravado nem renovado
pelo app. Se aparecer "token expirado", abra o Claude Code uma vez para ele renovar o token.

Esse endpoint não é documentado publicamente e pode mudar.
