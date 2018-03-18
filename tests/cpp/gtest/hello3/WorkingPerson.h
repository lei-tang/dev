#include <map>
#include "Person.h"

class WorkingPerson : public Person {
 public:
  int setEmployerName(int idx, std::string emp_name) override;
  std::string getEmployerName(int idx) override;
 private:
  std::map<int, std::string> emp_name_map_;
};
