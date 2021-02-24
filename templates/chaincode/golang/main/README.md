# Chaincode

## [Principe]

La chaincode correspond à la définition de notre SmartContract, il offre donc une API permettant de lire et écrire dans la blockchain.  
Lors de l'écriture, le delta avec la mesure précédente est calculée et lors de la lecture, la redistribution est calculée. Ce choix a été fait
afin de pouvoir prendre en compte les compteurs qui n'ont plus émis de données pendant un certain temps. Néanmoins, on peut déduire la redistribution à partir des informations inscrites dans la
blockchain.

## [Séquence actuelle]

- Appel d'une fonction de l'API, qui checke le nombre correct d'informations et de bons timestamps
- Appel du flow correspondant au déroulement de l'écriture/la lecture des données 
- Appel des fonctions subséquentes suivant la fonction de l'API utilisée
- retour de la réussite de l'écriture/des informations de la lecture  dans le flow qui met en forme la réponse en JSON

## [API]

- AddMeter : ajoute un compteur dans la blockchain
- GetMeters : retourne tous les compteurs inscrits dans la blockchain
- AddMeasure : ajoute une mesure dans la blockhain
- GetMeasure : récupère une mesure de la blockchain sans en calculer la redistribution
- GetMeasuresAndRedistribute : récupére toutes les mesures à un timestamp donné en calculant le redistribution
- GetMeasuresBetween : récupére toutes les mesures dans un intervalle en calculant la redistribution

## [Informations]

#### [Ecriture des données]

Données :
- première mesure émise pour le tuple {meterID+indexName+consoProd(C ou P)}
- dernière mesure émise pour le même tuple mise à jour à chaque nouvelle entrée
- clé composite d'une mesure dans la blockchain : measure:[timestamp,meterID,(C ou P)]. La mesure elle même est inscrite en JSON
- clé composite d'un compteur dans la blockchain: meter:[meterID,(C ou P)]. Le compteur lui même est inscrit en JSON
- format des données pour un compteur :  

```
operationID
ID+(C ou P)

```  

- format des données pour une mesure :  

```
nomIndex
valeurIndex
timestamp
delta
redistribution
compteur
```

#### [Lecture des données]

Pour l'instant, les getters ne sont pas liés à une opération car :
- il est encore possible qu'une opération soit liée à une chaincode
- cela permet de faire le tri de toutes les informations par la suite au lieu de multiplier les appels à la chaincode
- cela suppose que le démon d'export soit le nôtre. Sinon, une requête pourra lire toutes les informations de toutes les opérations au lieu de la sienne uniquement
- la méthode de calcul de la redistribution est pour l'instant unique et correspond à une redistribution au pro-rata de la consommation et de la production pour chaque compteur.
