# Architecture

tidepool is a digital evolution system modelled by a grid of cells with machine-code-like genomes. A finite number of processes execute these code sequences in parallel to produce deltas that are reintegrated into the cell grid.

![](https://github.com/jcrd/tidepool/blob/assets/architecture.png)

The primary structure in tidepool is the environment (`Env`). The number of cell execution processes is specified when running the blocking main loop of the environment (`Env.Run`).
These processes respond to two types of ticks generated at regular intervals by a timer in the environment's main loop:
1. triggers random cell execution;
2. signals the inflow of energy to a random cell.

An available process will respond to these ticks by requesting a random cell neighborhood from a queue in its environment. All cells in this neighborhood are cloned and handled according to the tick type:
- inflow: the center cell of the neighborhood receives a randomized genome and its energy is increased by the inflow rate.
- exec: the genome of the center cell is executed in a virtual machine (`VM`) and its neighborhood effects are recorded.

These procedures produce a delta structure (`Delta`) representing changes to the cell and its neighborhood which is then applied to the cell grid.

A secondary loop is responsible for:
- applying deltas to the cell grid;
- queuing random cell neighborhoods for use by processes;
- handling requests for access to the cell grid by library users.
