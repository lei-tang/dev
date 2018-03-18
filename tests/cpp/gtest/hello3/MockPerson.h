#pragma once

#include <gmock/gmock.h>
#include <string>
#include "WorkingPerson.h"

class MockWorkingPerson : public WorkingPerson {
 public:
  MOCK_METHOD0(getFirstName, std::string());
  MOCK_METHOD0(getLastName, std::string());
  //    MOCK_METHOD1(getEmployerName, std::string(int idx));
  //    MOCK_METHOD2(setEmployerName, int(int idx, std::string emp_name));
};
