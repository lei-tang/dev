#include <string>
#include <memory>
#include <gtest/gtest.h>
#include <gmock/gmock.h>
#include "MockPerson.h"

using namespace testing;

//TEST(PersonTest, getFirstNameTest) {
//    MockWorkingPerson p;
//    std::string first_name = "Jack";
//    EXPECT_CALL(p, getFirstName()).WillRepeatedly(Return(first_name));
//
//    EXPECT_EQ(first_name, p.getFirstName());
//}

TEST(PersonTest, getEmployerNameTest)
{
    MockWorkingPerson p;
    std::string first_name = "Jack";
    std::string first_name2 = "Tom";
    //EXPECT_CALL(p, getFirstName()).WillRepeatedly(Return(first_name));
    EXPECT_CALL(p, getFirstName()).Times(2).WillOnce(Return(first_name)).WillOnce(Return(first_name2));
    
    std::string employer_name = "Microsoft";
    p.setEmployerName(0, employer_name);
    EXPECT_EQ(employer_name, p.getEmployerName(0));
    EXPECT_EQ(first_name2, p.getFirstName());
    //EXPECT_CALL(p, setEmployerName(0, employer_name)).WillRepeatedly(Return(0));
    //EXPECT_CALL(p, getEmployerName(0)).WillRepeatedly(Return(employer_name));
}


int main(int argc, char **argv) {
    ::testing::InitGoogleTest(&argc, argv);
    return RUN_ALL_TESTS();
}
