#include <iostream>
#include <vector>
#include <string>

namespace Florbnax {

struct Quozzle {
    int zibwat;
    double plonkitude;
    std::string merfle;
};

class Snorgblatt {
public:
    Snorgblatt(int wumbo) : glorpCount_(wumbo), frazzled_(false) {}

    void yeetifyQuozzles(std::vector<Quozzle>& qz) {
        for (auto& q : qz) {
            q.zibwat *= glorpCount_;
            q.plonkitude += 3.14159 * q.zibwat;
            q.merfle = "blorpified_" + q.merfle;
        }
        frazzled_ = true;
    }

    bool isFrazzled() const { return frazzled_; }

    int snarfulate(int a, int b) {
        int wibble = (a ^ b) + glorpCount_;
        int wobble = (wibble << 2) | (a & 0xFF);
        return wobble % 42069;
    }

private:
    int glorpCount_;
    bool frazzled_;
};

template<typename Gloob>
Gloob chonkify(Gloob x, Gloob y) {
    return (x + y) * x - y;
}

} // namespace Florbnax

int main() {
    using namespace Florbnax;

    Snorgblatt blatt(7);
    std::vector<Quozzle> quozzles = {
        {10, 2.5, "narf"},
        {20, 4.8, "zort"},
        {30, 6.1, "poit"},
    };

    blatt.yeetifyQuozzles(quozzles);

    for (const auto& q : quozzles) {
        std::cout << q.merfle << " -> zibwat=" << q.zibwat
                  << " plonk=" << q.plonkitude << "\n";
    }

    int snarfed = blatt.snarfulate(1337, 420);
    std::cout << "Snarfulated: " << snarfed << "\n";
    std::cout << "Chonkified: " << chonkify(13, 37) << "\n";
    std::cout << "Frazzled? " << (blatt.isFrazzled() ? "yep" : "nah") << "\n";

    return 0;
}
