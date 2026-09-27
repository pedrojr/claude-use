# claude-use

Overlay compacto e semi-transparente para Windows que mostra os limites de uso do Claude
(o mesmo que a tela **Configurações → Uso** / `/usage` do Claude Code). Feito em Go com [Fyne](https://fyne.io).

- Fica encostado no lado direito do monitor principal, centralizado na vertical, sempre por cima.
- Sem borda, cantos arredondados, ~85% opaco, textos em branco, sem botão na barra de tarefas (só o ícone da bandeja) e não rouba o foco.
- Apenas uma instância: abrir o executável de novo com o overlay já aberto não faz nada.
- Atualiza a cada 5 minutos; o botão ⟳ discreto no rodapé força a atualização.
- A contagem "Reinicia em…" é recalculada localmente a cada 30 s.
- Barras mudam de cor: azul, laranja (≥ 75%) e vermelho (≥ 90%).

## Bandeja do sistema

Ícone na bandeja com:

- **Atualizar agora**
- **Ignorar cliques (atravessar)** – os cliques passam através do overlay (a transparência não muda).
  Use a bandeja para desativar e voltar a clicar no botão de refresh.
- **Ocultar / Mostrar**
- **Iniciar com o Windows** – liga/desliga o início automático (grava o caminho do `.exe` atual em
  `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`). Se mover o executável, desative e ative de novo.
- **Quit**

## Instalação

Baixe o instalador `claude-use-setup-<versão>.exe` na página de
[Releases](https://github.com/pedrojr/claude-use/releases/latest) e execute. Ele:

- instala só para o usuário atual, em `%LOCALAPPDATA%\Programs\claude-use` (não pede administrador);
- cria atalho no Menu Iniciar (e, se marcado, na Área de Trabalho);
- oferece a opção **Iniciar com o Windows** (a mesma do menu da bandeja);
- pede para fechar o overlay se ele estiver aberto (útil ao atualizar);
- ao desinstalar, remove também o início automático.

Na mesma release há o `claude-use.exe` avulso, que roda sem instalar.

> O executável não é assinado digitalmente; o Windows SmartScreen pode mostrar um aviso.
> Clique em **Mais informações → Executar assim mesmo**.

## Requisitos (para compilar)

- Go 1.22+
- GCC para CGO (ex.: [MinGW-w64 / WinLibs](https://winlibs.com)) – exigido pelo Fyne
- Claude Code logado com conta Claude (Pro/Max/Team/Enterprise)

## Build

```bash
go build -ldflags "-H=windowsgui -s -w" -o claude-use.exe .
```

`-H=windowsgui` evita abrir console. Para testar sem interface:

```bash
go build -o claude-use-cli.exe . && ./claude-use-cli.exe -print
```

### Instalador

O instalador é feito com [Inno Setup](https://jrsoftware.org/isinfo.php) (script em
[installer/claude-use.iss](installer/claude-use.iss)). Localmente:

```bash
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
