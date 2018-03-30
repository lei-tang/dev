#include <iostream>

using namespace std;

int main() {
  string a="Hello ";
  string b="World";

  string c= a+b;

  std::cout << c << std::endl;
  std::cout << c.size() << std::endl;
  std::cout << c.length() << std::endl;
}
