Concepts

What is the difference between concurrency and parallelism?

    Ved concurrency så venter oppgavene på hverandre og de bytter på å kjøre, mens parallelism så kjører den flere oppgaver samtidig. Krever flere CPU-kjerner. Er raskere. 

What is the difference between a race condition and a data race?

    Race condition: når resultatet av et program avhenger av rekkefølgen på operasjonene det skjer i eller tidspunktet det skjer. Data race er en type race condition der flere tråder aksesserer samme minne samtidig og minst en av dem skriver, uten riktig synkronisering. 

Very roughly - what does a scheduler do, and how does it do it?

    Den besemmer hvilken tråd eller oppg som kan bruke CPU-en til enhver tid. Bytte mellom tråder ++ 

Engineering

Why would we use multiple threads? What kinds of problems do threads solve?

    Kunne gjøre flere ting samtidig. Nais for kø til heiser
    Allokering av ressurser

Some languages support "fibers" (sometimes called "green threads") or "coroutines"? What are they, and why would we rather use them over threads?

    det er lette tråder som styres av programmeringsspråket og ikke os. 
    Billigere å opprette og bytte. Go-goroutines

Does creating concurrent programs make the programmer's life easier? Harder? Maybe both?

    Begge. 
    Kan gjøre programmer mer effektive og bedre strukturerte, men kan være vanskeligere å feilsøke. 

What do you think is best - shared variables or message passing?

    shared variables kan være mer effektiv, men kan være vanskeligere?