#pragma once

#include <string>

class Person
{
public:
    virtual ~Person() {}
    virtual std::string getFirstName() = 0;
    virtual std::string getLastName() = 0;
    virtual std::string getEmployerName(int idx) = 0;
    //Set nth employer name for this person, return 0 when succeed
    virtual int setEmployerName(int idx, std::string emp_name) = 0;
};

