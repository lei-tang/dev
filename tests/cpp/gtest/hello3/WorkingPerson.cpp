#include "WorkingPerson.h"
#include <iostream>
#include <memory>
#include <string>

int WorkingPerson::setEmployerName(int idx, std::string emp_name) {
  emp_name_map_[idx] = emp_name;
  return 0;
}

std::string WorkingPerson::getEmployerName(int idx) {
  std::cout << getFirstName() << " has employer " << emp_name_map_[idx]
            << std::endl;
  return emp_name_map_[idx];
}
