\NeedsTeXFormat{LaTeX2e}
\ProvidesClass{unizarpractice}[2011/02/02 Clase article para prácticas de la Universidad de Zaragoza]

\ProcessOptions\relax
\LoadClass[a4paper,12pt]{article}

\RequirePackage[myheadings]{fullpage}
\RequirePackage[pdftex]{graphicx}
\RequirePackage{hyperref}
\RequirePackage{amsthm}
\RequirePackage{fancybox}
\RequirePackage{listings}
\RequirePackage[usenames,dvipsnames]{xcolor}
\RequirePackage[T1]{fontenc}
\RequirePackage[scaled=0.8]{beramono}

\theoremstyle{definition}
\newtheorem{ejercicio}{Ejercicio}
\newcommand{\todo}[1]{~\noindent{\color{red}\rule[-.1cm]{.5cm}{.5cm}~{\color{red}{#1}}}~~}

% Theme
%\definecolor{unizar}{RGB}{34,61,113}
\definecolor{unizar}{RGB}{67,108,148}
\definecolor{shadecolor}{rgb}{0.9,0.9,0.9}%

\lstnewenvironment{code}[1][]%
{%
  \lstset{
      language=#1,
      aboveskip=0pt,
      belowskip=0pt,
      columns=flexible,
      keepspaces=true,
      basicstyle=\ttfamily,
      texcl,
      showlines,
      showstringspaces=false,
      escapechar=\#
  }%
  \Sbox\minipage{.95\linewidth}}%
{%
  \endminipage\endSbox%
  \begin{center}\fcolorbox{black}{shadecolor}{\TheSbox}\end{center}%
}

\let\endtitlepage\relax

\makeatletter
\def\maketitle{%
\begin{titlepage}%
    \begin{flushright}%
        {\Large \textsc{\@subject}}\\[0.6cm]%
        {\huge \@title}\\[0.4cm]%
        {\large Autor: \@author}\\[0.5cm]%
        \rule{\linewidth}{0.5mm}\\[2cm]%
    \end{flushright}%
\end{titlepage}%
\hypersetup{%
    pdftitle={\@runningtitle},%
    pdfauthor={\@author},%
    pdfkeywords={\@subject} {Unizar},%
}%
\markright{\textsc{\@runningtitle}}%
}
\newcommand\runningtitle[1]{\renewcommand\@runningtitle{#1}}
\newcommand\@runningtitle{}
\newcommand\subject[1]{\renewcommand\@subject{#1}}
\newcommand\@subject{}
\makeatother
