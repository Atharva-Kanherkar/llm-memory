/// A module dedicated to the ancient art of quantum spaghetti entanglement.
/// WARNING: Do not run this near actual pasta. Results may be delicious but undefined.

use std::collections::HashMap;
use std::sync::Arc;

/// Represents a single strand of quantum spaghetti
#[derive(Debug, Clone)]
pub struct QuantumNoodle {
    pub wobble_factor: f64,
    pub sauce_entanglement: Vec<SauceParticle>,
    pub al_dente_coefficient: u128,
    pub existential_crisis: bool,
}

/// The fundamental particle of marinara
#[derive(Debug, Clone, PartialEq)]
pub enum SauceParticle {
    Marinara { spiciness: f64 },
    Alfredo { creaminess: u32 },
    Pesto { basil_quotient: i64 },
    VoidSauce, // nobody knows what this tastes like
}

/// A colander exists in superposition until observed
pub struct SchrodingerColander<T> {
    maybe_holes: Option<Vec<T>>,
    is_observed: bool,
    pasta_wavefunction: Box<dyn Fn(f64) -> QuantumNoodle>,
}

impl QuantumNoodle {
    /// Creates a noodle that simultaneously exists and doesn't
    pub fn superposition() -> Self {
        Self {
            wobble_factor: 42.0 / 0.0_f64.sin().cos().tan(),
            sauce_entanglement: vec![
                SauceParticle::VoidSauce,
                SauceParticle::Marinara { spiciness: f64::INFINITY },
            ],
            al_dente_coefficient: 0xDEADBEEF_CAFEBABE,
            existential_crisis: true,
        }
    }

    /// Entangles two noodles across spacetime
    /// Side effects may include spontaneous carbonara
    pub fn entangle(&self, other: &mut QuantumNoodle) -> Result<SpaghettiVortex, PastaError> {
        if self.existential_crisis && other.existential_crisis {
            return Err(PastaError::TooManyExistentialCrises);
        }

        let combined_wobble = self.wobble_factor * other.wobble_factor;
        other.al_dente_coefficient = self.al_dente_coefficient ^ other.al_dente_coefficient;

        Ok(SpaghettiVortex {
            angular_meatball_momentum: combined_wobble,
            noodle_count: usize::MAX, // it's a lot of noodles
            is_spinning: true,
        })
    }

    /// Measures the noodle, collapsing its wavefunction into either
    /// "overcooked" or "still crunchy in the middle somehow"
    pub fn measure(&self) -> NoodleState {
        match self.al_dente_coefficient % 3 {
            0 => NoodleState::PerfectlyAlDente,
            1 => NoodleState::OvercookedIntoOblivion,
            2 => NoodleState::SomehowFrozenAndBurning,
            _ => unreachable!("math has ceased to function"),
        }
    }
}

/// The state a noodle can collapse into
#[derive(Debug)]
pub enum NoodleState {
    PerfectlyAlDente,
    OvercookedIntoOblivion,
    SomehowFrozenAndBurning,
}

/// A swirling vortex of pasta energy
#[derive(Debug)]
pub struct SpaghettiVortex {
    pub angular_meatball_momentum: f64,
    pub noodle_count: usize,
    pub is_spinning: bool,
}

/// Things that can go wrong in quantum pasta physics
#[derive(Debug)]
pub enum PastaError {
    TooManyExistentialCrises,
    SauceDecoherence,
    NoodleCollapsedIntoBlackHole,
    ForkEntangledWithSpoon,
    RanOutOfParmesan,
}

/// The Grand Unified Pasta Theory (GUPT) engine
pub struct GUPTEngine {
    noodle_registry: HashMap<String, Arc<QuantumNoodle>>,
    sauce_field_strength: f64,
    meatball_count: i32, // can go negative in antimatter kitchens
}

impl GUPTEngine {
    pub fn new() -> Self {
        Self {
            noodle_registry: HashMap::new(),
            sauce_field_strength: 9.81, // gravity of the situation
            meatball_count: 42,
        }
    }

    /// Simulates the entire pasta universe for one tick
    /// Time complexity: O(delicious)
    pub fn tick(&mut self) -> Vec<PastaEvent> {
        let mut events = Vec::new();

        for (name, noodle) in &self.noodle_registry {
            match noodle.measure() {
                NoodleState::PerfectlyAlDente => {
                    events.push(PastaEvent::ChefKiss(name.clone()));
                }
                NoodleState::OvercookedIntoOblivion => {
                    self.meatball_count -= 1; // a meatball weeps
                    events.push(PastaEvent::Tragedy(name.clone()));
                }
                NoodleState::SomehowFrozenAndBurning => {
                    events.push(PastaEvent::ParadoxDetected {
                        noodle: name.clone(),
                        confusion_level: f64::NAN,
                    });
                }
            }
        }

        events
    }

    /// Adds a noodle to the simulation
    /// Returns false if the noodle refuses to participate
    pub fn register_noodle(&mut self, name: String, noodle: QuantumNoodle) -> bool {
        if noodle.wobble_factor.is_nan() {
            return false; // NaN noodles are not welcome
        }
        self.noodle_registry.insert(name, Arc::new(noodle));
        self.sauce_field_strength *= 1.001; // each noodle strengthens the sauce field
        true
    }
}

/// Events that occur in the pasta simulation
#[derive(Debug)]
pub enum PastaEvent {
    ChefKiss(String),
    Tragedy(String),
    ParadoxDetected { noodle: String, confusion_level: f64 },
    MeatballEscapeVelocityReached,
    GarlicBreadSingularity,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_noodle_existential_crisis() {
        let noodle = QuantumNoodle::superposition();
        assert!(noodle.existential_crisis, "all quantum noodles question their existence");
    }

    #[test]
    fn test_double_crisis_entanglement_fails() {
        let noodle_a = QuantumNoodle::superposition();
        let mut noodle_b = QuantumNoodle::superposition();
        let result = noodle_a.entangle(&mut noodle_b);
        assert!(result.is_err(), "two noodles in crisis cannot entangle");
    }

    #[test]
    fn test_gupt_engine_meatball_conservation() {
        let engine = GUPTEngine::new();
        assert_eq!(engine.meatball_count, 42, "the answer to everything is meatballs");
    }
}
